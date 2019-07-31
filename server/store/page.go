package store

import (
	"context"

	"cloud.google.com/go/datastore"
	"github.com/seaptc/server/model"
)

const pageKind = "page"

func pageKey(path string) *datastore.Key {
	return datastore.NameKey(pageKind, path, conferenceEntityGroupKey)
}

func (store *Store) SetPage(ctx context.Context, page *model.Page) error {
	_, err := store.dsClient.Put(ctx, pageKey(page.Path), page)
	return err
}

func (store *Store) GetPage(ctx context.Context, path string) (*model.Page, error) {
	var page model.Page
	return (*model.Page)(&page), store.dsClient.Get(ctx, pageKey(path), &page)
}

func (store *Store) GetPageHashes(ctx context.Context) (map[string]string, error) {
	var pages []*model.Page
	// no ancestor in query for use of built-in index.
	_, err := store.dsClient.GetAll(ctx, datastore.NewQuery(pageKind).Project(model.Page_Hash), &pages)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, page := range pages {
		result[page.Path] = page.Hash
	}
	return result, nil
}
