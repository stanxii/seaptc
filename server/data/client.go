package data

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

//go:generate go run gogen.go -output fields.go

func NotFoundOK(ds *firestore.DocumentSnapshot, err error) (*firestore.DocumentSnapshot, error) {
	if grpc.Code(err) == codes.NotFound {
		err = nil
	}
	return ds, err
}

func GetDocTo(ctx context.Context, client *firestore.Client, path string, v interface{}) error {
	sn, err := NotFoundOK(client.Doc(path).Get(ctx))
	if err != nil {
		return err
	}
	return sn.DataTo(v)
}
