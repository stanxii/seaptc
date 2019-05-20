package store

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/seaptc/server/model"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	appConfigPath    = "misc/appconfig"
	conferencePath   = "misc/conference"
	classesPath      = "classes"
	participantsPath = "participants"
)

func classPath(c *model.Class) string {
	return fmt.Sprintf("%s/%d", classesPath, c.Number)
}

func participantPath(p *model.Participant) string {
	ya := "a"
	if p.Youth {
		ya = "y"
	}
	return fmt.Sprintf("%s/%s_%s_%s_%s_%s", participantsPath, p.LastName, p.FirstName, p.Suffix, ya, p.RegistrationNumber)
}

func IsNotFoundError(err error) bool {
	return grpc.Code(err) == codes.NotFound
}

func notFoundOK(ds *firestore.DocumentSnapshot, err error) (*firestore.DocumentSnapshot, error) {
	if IsNotFoundError(err) {
		err = nil
	}
	return ds, err
}

var (
	projectID        string
	useEmulator      bool
	setupFlagsCalled bool
)

func SetupFlags() {
	flag.StringVar(&projectID, "project", "seaptc", "Project for Firestore")
	flag.BoolVar(&useEmulator, "dsemul", os.Getenv("GAE_INSTANCE") == "", "Use Firestore emulator")
	setupFlagsCalled = true
}

type Store struct {
	fsClient *firestore.Client

	classes struct {
		mu           sync.Mutex
		value        map[int]*model.Class
		maxUpdateTime time.Time
        readTime time.Time
	}
}

// NewFromFlags creates a client using flags defined in this package.
func NewFromFlags(ctx context.Context) (*Store, error) {
	if !setupFlagsCalled {
		return nil, errors.New("store.SetupFlags not called")
	}

	const emulatorKey = "FIRESTORE_EMULATOR_HOST"
	if useEmulator {
		if os.Getenv(emulatorKey) == "" {
			return nil, fmt.Errorf("firestore emulator host not set. export %s=host:port", emulatorKey)
		}
	} else {
		os.Unsetenv(emulatorKey)
	}

	fsClient, err := firestore.NewClient(ctx, projectID)
	return &Store{fsClient: fsClient}, err
}

func (store *Store) getDocTo(ctx context.Context, path string, v interface{}) error {
	sn, err := notFoundOK(store.fsClient.Doc(path).Get(ctx))
	if err != nil {
		return err
	}
	return sn.DataTo(v)
}

func (store *Store) GetAppConfig(ctx context.Context) (*model.AppConfig, error) {
	var config model.AppConfig
	return &config, store.getDocTo(ctx, appConfigPath, &config)
}

func (store *Store) SetAppConfig(ctx context.Context, config *model.AppConfig) error {
	_, err := store.fsClient.Doc(appConfigPath).Set(ctx, config)
	return err
}

func (store *Store) GetConference(ctx context.Context) (*model.Conference, error) {
	var conf model.Conference
	return &conf, store.getDocTo(ctx, conferencePath, &conf)
}

func (store *Store) SetConference(ctx context.Context, conf *model.Conference) error {
	_, err := store.fsClient.Doc(conferencePath).Set(ctx, conf)
	return err
}

func (store *Store) Foo() *firestore.Client { return store.fsClient }

func (store *Store) GetClasses(ctx context.Context, maxAge time.Duration) (map[int]*model.Class, error) {
	store.classes.mu.Lock()
	defer store.classes.mu.Unlock()

	// TODO: modify to handle arbitrary queries and value types
	// TODO: use cookie to track high water mark

	if time.Since(store.classes.readTime) < maxAge {
		return store.classes.value, nil
	}

	snaps, err := store.fsClient.Collection(classesPath).
		Where(model.Class_Timestamp, ">", store.classes.maxUpdateTime).
		Documents(ctx).
		GetAll()

	if err != nil {
		return nil, err
	}

	if len(snaps) == 0 {
		log.Printf("GetClasses: n=0")
		store.classes.readTime = time.Now()
		return store.classes.value, nil
	}

	classes := make(map[int]*model.Class)
	for n, c := range store.classes.value {
		classes[n] = c
	}

	var maxUpdateTime time.Time
	for _, snap := range snaps {
		var c model.Class
		if err := snap.DataTo(&c); err != nil {
			return nil, err
		}
        log.Println(c.Number, snap.UpdateTime.UnixNano(), c.LastUpdateTime.UnixNano())
		classes[c.Number] = &c
        if snap.UpdateTime.After(maxUpdateTime) {
			maxUpdateTime = snap.UpdateTime
		}
	}
	store.classes.readTime = time.Now()
	store.classes.maxUpdateTime = maxUpdateTime
	store.classes.value = classes

	log.Printf("GetClasses: n=%d, maxTimeStamp=%d", len(snaps), maxUpdateTime.UnixNano())

	return classes, nil
}

func (store *Store) UpdateClassesFromSheet(ctx context.Context, sheetClasses []*model.Class) (int, error) {
	storeClasses, err := store.GetClasses(ctx, 0)
	if err != nil {
		return 0, err
	}

    store.classes.mu.Lock()
    maxUpdateTime := store.classes.maxUpdateTime
    store.classes.mu.Unlock()

	batch := store.fsClient.Batch()
	updateCount := 0
	for _, sheetClass := range sheetClasses {
		if storeClass := storeClasses[sheetClass.Number]; storeClass == nil ||
			!storeClass.EqualSheetFields(sheetClass) ||
            storeClass.LastUpdateTime.After(maxUpdateTime) {
			m := sheetClass.SheetFields()
			m[model.Class_Timestamp] = firestore.ServerTimestamp
			batch.Set(store.fsClient.Doc(classPath(sheetClass)), m, firestore.MergeAll)
			updateCount++
		}
	}

/*
    for _, storeClass := range storeClasses {
        if _, ok := sheetClasses[storeClass.Number]; !ok {
            batch.Delete(store.fsClient.Doc(classPath(storeClass)), nil)
            updateCount++
        }
    }
*/

	if updateCount == 0 {
		return 0, nil
	}

	results, err := batch.Commit(ctx)
	if err != nil {
		return 0, err
	}
	for _, result := range results {
		log.Printf("Update: %d", result.UpdateTime.UnixNano())
	}
	return updateCount, nil
}

func (store *Store) GetClass(ctx context.Context, number string) (*model.Class, error) {
	var class model.Class
	return &class, store.getDocTo(ctx, classesPath+"/"+number, &class)
}
