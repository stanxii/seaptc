package store

import (
	"context"

	"cloud.google.com/go/datastore"
	"github.com/seaptc/server/model"
)

func miscKey(kind string) *datastore.Key {
	return datastore.IDKey(kind, 1, conferenceEntityGroupKey)
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
	_, err := store.dsClient.Put(ctx, miscKey("conference"), conf)
	return err
}

type suggestedSchedules struct {
	SuggestedSchedules []*model.SuggestedSchedule `datastore:"suggestedSchedules,noindex"`
}

func (store *Store) GetSuggestedSchedules(ctx context.Context) ([]*model.SuggestedSchedule, error) {
	var ss suggestedSchedules
	err := noEntityOK(store.dsClient.Get(ctx, miscKey("suggestedSchedules"), &ss))
	return ss.SuggestedSchedules, err
}

func (store *Store) SetSuggestedSchedules(ctx context.Context, ss []*model.SuggestedSchedule) error {
	_, err := store.dsClient.Put(ctx, miscKey("suggestedSchedules"), &suggestedSchedules{ss})
	return err
}
