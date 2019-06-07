package store

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/datastore"
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
	dsClient *datastore.Client
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

func saveInts(vs []int) string {
	var buf []byte
	for i, v := range vs {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = strconv.AppendInt(buf, int64(v), 10)
	}
	return string(buf)
}

func loadInts(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	vs := make([]int, len(parts))
	for i, part := range parts {
		v, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		vs[i] = v
	}
	return vs, nil
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
