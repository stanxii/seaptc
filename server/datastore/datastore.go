package datastore

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type Client struct {
	*firestore.Client
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

// NewClient creates a client using flags defined in this package.
func NewFromFlags(ctx context.Context) (*Client, error) {
	if !setupFlagsCalled {
		return nil, errors.New("datastore.SetupFlags not called")
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
	return &Client{fsClient}, err
}

func (c *Client) GetDocTo(ctx context.Context, path string, v interface{}) error {
	sn, err := NotFoundOK(c.Doc(path).Get(ctx))
	if err != nil {
		return err
	}
	return sn.DataTo(v)
}

func NotFoundOK(ds *firestore.DocumentSnapshot, err error) (*firestore.DocumentSnapshot, error) {
	if grpc.Code(err) == codes.NotFound {
		err = nil
	}
	return ds, err
}
