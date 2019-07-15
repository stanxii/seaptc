package store

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/seaptc/server/model"
)

var (
	projectID        string
	useEmulator      bool
	setupFlagsCalled bool
)

func SetupFlags() {
	flag.StringVar(&projectID, "project", "seaptc-ds", "Project for Datastore")
	flag.BoolVar(&useEmulator, "emul", os.Getenv("GAE_INSTANCE") == "", "Use Datastore emulator")
	setupFlagsCalled = true
}

type Store struct {
	dsClient        *datastore.Client
	classInfoCache  valueCache
	conferenceCache valueCache
}

// NewFromFlags creates a client using flags defined in this package.
func NewFromFlags(ctx context.Context) (*Store, error) {
	if !setupFlagsCalled {
		return nil, errors.New("store.SetupFlags not called")
	}

	const emulatorKey = "DATASTORE_EMULATOR_HOST"
	if useEmulator {
		if os.Getenv(emulatorKey) == "" {
			return nil, fmt.Errorf("Datatstore emulator host not set.\n"+
				"To start the emulator run: gcloud beta emulators datastore start\n"+
				"and export %s=host:port", emulatorKey)
		}
	} else {
		os.Unsetenv(emulatorKey)
	}

	dsClient, err := datastore.NewClient(ctx, projectID)
	return &Store{dsClient: dsClient}, err
}

var ErrNotFound = datastore.ErrNoSuchEntity

func noEntityOK(err error) error {
	if err == datastore.ErrNoSuchEntity {
		return nil
	}
	if errs, ok := err.(datastore.MultiError); ok {
		for _, err := range errs {
			if err != nil && err != datastore.ErrNoSuchEntity {
				return errs
			}
		}
		return nil
	}
	return err
}

func filterProperties(ps []datastore.Property, deleted map[string]bool) []datastore.Property {
	i := 0
	for _, p := range ps {
		if deleted[p.Name] {
			continue
		}
		ps[i] = p
		i++
	}
	return ps[:i]
}

var conferenceEntityGroupKey = datastore.IDKey("conference", 1, nil)

var (
	errNoUpdate = errors.New("no update")
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

func checkUpdateFunc(update interface{}) (reflect.Value, reflect.Type, error) {
	updatev := reflect.ValueOf(update)
	t := updatev.Type()
	var err error
	if t.Kind() != reflect.Func ||
		t.NumIn() != 1 ||
		t.NumOut() != 1 ||
		t.Out(0) != errorType ||
		t.In(0).Kind() != reflect.Ptr {
		err = errors.New("update func not f(v *Type) error")
	}
	return updatev, t, err
}

func (store *Store) updateEntity(ctx context.Context, key *datastore.Key, update interface{}) error {
	updatev, t, err := checkUpdateFunc(update)
	if err != nil {
		return err
	}
	_, err = store.dsClient.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		dst := reflect.New(t.In(0).Elem())
		err := noEntityOK(tx.Get(key, dst.Interface()))
		if err != nil {
			return err
		}
		out := updatev.Call([]reflect.Value{dst})
		err, _ = out[0].Interface().(error)
		if err == errNoUpdate {
			return nil
		}
		if err != nil {
			return err
		}
		_, err = tx.Put(key, dst.Interface())
		return err
	})
	return err
}

func (store *Store) updateEntities(ctx context.Context, keys []*datastore.Key, update interface{}) (int, error) {
	updatev, t, err := checkUpdateFunc(update)
	if err != nil {
		return 0, err
	}

	var mutationCount int

	for len(keys) > 0 {
		n := len(keys)
		if n > 50 {
			n = 50
		}
		var txMutationCount int
		_, err := store.dsClient.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
			dst := reflect.MakeSlice(reflect.SliceOf(t.In(0)), n, n)
			err := noEntityOK(tx.GetMulti(keys[:n], dst.Interface()))
			if err != nil {
				return err
			}
			var mutations []*datastore.Mutation
			for i := 0; i < n; i++ {
				elem := dst.Index(i)
				if elem.IsNil() {
					continue
				}
				out := updatev.Call([]reflect.Value{elem})
				err, _ = out[0].Interface().(error)
				if err == errNoUpdate {
					continue
				}
				if err != nil {
					return err
				}
				mutations = append(mutations, datastore.NewUpdate(keys[i], elem.Interface()))
			}
			txMutationCount = len(mutations)
			if txMutationCount != 0 {
				_, err = tx.Mutate(mutations...)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return mutationCount, err
		}
		mutationCount += txMutationCount
		keys = keys[n:]
	}
	return mutationCount, nil
}

type valueCache struct {
	mu      sync.RWMutex
	updated time.Time
	value   interface{}
}

func (vc *valueCache) clear() {
	vc.mu.Lock()
	vc.updated = time.Time{}
	vc.mu.Unlock()
}

func (vc *valueCache) get(ctx context.Context, maxAge time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	vc.mu.RLock()
	updated := vc.updated
	value := vc.value
	vc.mu.RUnlock()
	if time.Since(updated) < maxAge {
		return value, nil
	}

	vc.mu.Lock()
	defer vc.mu.Unlock()

	if time.Since(vc.updated) < maxAge {
		return vc.value, nil
	}

	value, err := fn()
	if err != nil {
		return value, err
	}

	vc.updated = time.Now()
	vc.value = value
	return value, nil
}

func (store *Store) GetCachedConference(ctx context.Context) (*model.Conference, error) {
	v, err := store.conferenceCache.get(ctx, 10*time.Minute, func() (interface{}, error) {
		v, err := store.GetConference(ctx)
		return v, err
	})
	conf, _ := v.(*model.Conference)
	return conf, err
}

func (store *Store) GetCachedClassInfo(ctx context.Context) (*model.ClassInfo, error) {
	v, err := store.classInfoCache.get(ctx, 15*time.Minute, func() (interface{}, error) {
		classes, err := store.GetAllClasses(ctx)
		return model.NewClassInfo(classes), err
	})
	cms, _ := v.(*model.ClassInfo)
	return cms, err
}
