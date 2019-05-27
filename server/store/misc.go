package store

import (
	"context"

	"cloud.google.com/go/datastore"
	"github.com/seaptc/server/model"
)

const miscKind = "misc"

func miscKey(what string) *datastore.Key {
	return datastore.NameKey(miscKind, what, nil)
}

func (store *Store) GetAppConfig(ctx context.Context) (*model.AppConfig, error) {
	var config model.AppConfig
	return &config, noEntityOK(store.dsClient.Get(ctx, miscKey("appConfig"), &config))
}

func (store *Store) SetAppConfig(ctx context.Context, config *model.AppConfig) error {
	_, err := store.dsClient.Put(ctx, miscKey("appConfig"), config)
	return err
}

func (store *Store) GetConference(ctx context.Context) (*model.Conference, error) {
	var conf model.Conference
	return &conf, noEntityOK(store.dsClient.Get(ctx, miscKey("conference"), &conf))
}

func (store *Store) SetConference(ctx context.Context, conf *model.Conference) error {
	conf.SuggestedSchedules = ""
	_, err := store.dsClient.Put(ctx, miscKey("conference"), conf)
	return err
}
