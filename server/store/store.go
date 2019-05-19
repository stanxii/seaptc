package store

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/seaptc/server/data"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	appConfigPath    = "misc/appconfig"
	classesPath      = "classses"
	participantsPath = "participants"
)

func classPath(c *data.Class) string {
	return fmt.Sprintf("%s/%d", classesPath, c.Number)
}

func participantPath(p *data.Participant) string {
	ya := "a"
	if p.Youth {
		ya = "y"
	}
	return fmt.Sprintf("%s/%s_%s_%s_%s_%s", participantsPath, p.LastName, p.FirstName, p.Suffix, ya, p.RegistrationNumber)
}

func notFoundOK(ds *firestore.DocumentSnapshot, err error) (*firestore.DocumentSnapshot, error) {
	if grpc.Code(err) == codes.NotFound {
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
	return &Store{fsClient}, err
}

func (store *Store) getDocTo(ctx context.Context, path string, v interface{}) error {
	sn, err := notFoundOK(store.fsClient.Doc(path).Get(ctx))
	if err != nil {
		return err
	}
	return sn.DataTo(v)
}

func (store *Store) GetAppConfig(ctx context.Context) (*data.AppConfig, error) {
	var config data.AppConfig
	return &config, store.getDocTo(ctx, appConfigPath, &config)
}

func (store *Store) SetAppConfig(ctx context.Context, config *data.AppConfig) error {
	_, err := store.fsClient.Doc(appConfigPath).Set(ctx, config)
	return err
}

type Classes struct {
	Slice []*data.Class       // Sorted by class number
	Map   map[int]*data.Class // Key is class number
}

func (store *Store) GetClasses(ctx context.Context) (*Classes, error) {
	snaps, err := store.fsClient.Collection(classesPath).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	classes := Classes{Map: make(map[int]*data.Class), Slice: make([]*data.Class, len(snaps))}

	for i, snap := range snaps {
		var c data.Class
		if err := snap.DataTo(&c); err != nil {
			return nil, err
		}
		classes.Map[c.Number] = &c
		classes.Slice[i] = &c
	}
	sort.Slice(classes.Slice, func(i, j int) bool { return classes.Slice[i].Number < classes.Slice[j].Number })
	return &classes, nil
}

func (store *Store) UpdateClasses(ctx context.Context, classes []*data.Class, fields func(*data.Class) map[string]interface{}) error {
	for _, class := range classes {
		_, err := store.fsClient.Doc(classPath(class)).Set(ctx, fields(class), firestore.MergeAll)
		if err != nil {
			return err
		}
	}
	return nil
}
