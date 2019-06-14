package store

import (
	"context"
	"fmt"

	"github.com/seaptc/server/model"

	"cloud.google.com/go/datastore"
)

const classKind = "class"

func classKey(number int) *datastore.Key {
	return datastore.IDKey(classKind, int64(number), conferenceEntityGroupKey)
}

// classπImportHashLoginCode is as destination type for project(import hash)
// queries.
type classΠImportHash struct {
	ImportHash string `datastore:"importHash"`
}

// xClass overrides datastore load and save on an model.Class.
type xClass model.Class

var deletedClassFields = map[string]bool{
	"dkNeedsUpdate": true,
	"titleNotes":    true,
	"number":        true,
}

func (c *xClass) Load(ps []datastore.Property) error {
	err := datastore.LoadStruct((*model.Class)(c), filterProperties(ps, deletedClassFields))
	if err != nil {
		return err
	}
	(*model.Class)(c).Init()
	return nil
}

func (c *xClass) LoadKey(k *datastore.Key) error {
	c.Number = int(k.ID)
	return nil
}

func (c *xClass) Save() ([]datastore.Property, error) {
	ps, err := datastore.SaveStruct((*model.Class)(c))
	return ps, err
}

func (store *Store) GetClass(ctx context.Context, number int) (*model.Class, error) {
	if !model.IsValidClassNumber(number) {
		return nil, ErrNotFound
	}
	var c xClass
	err := store.dsClient.Get(ctx, classKey(number), &c)
	return (*model.Class)(&c), err
}

var allClassesQuery = datastore.NewQuery(classKind).Ancestor(conferenceEntityGroupKey).Project(
	model.Class_Length,
	model.Class_Title,
	model.Class_Capacity,
	model.Class_Location,
	model.Class_Responsibility)

func (store *Store) GetAllClasses(ctx context.Context) ([]*model.Class, error) {
	var xclasses []*xClass
	_, err := store.dsClient.GetAll(ctx, allClassesQuery, &xclasses)
	if err != nil {
		return nil, err
	}
	classes := make([]*model.Class, len(xclasses))
	for i, xc := range xclasses {
		classes[i] = (*model.Class)(xc)
	}
	return classes, nil
}

func (store *Store) GetAllClassesFull(ctx context.Context) ([]*model.Class, error) {
	var xclasses []*xClass
	_, err := store.dsClient.GetAll(ctx, datastore.NewQuery(classKind).Ancestor(conferenceEntityGroupKey), &xclasses)
	if err != nil {
		return nil, err
	}
	classes := make([]*model.Class, len(xclasses))
	for i, xc := range xclasses {
		classes[i] = (*model.Class)(xc)
	}
	return classes, nil
}

func (store *Store) ImportClasses(ctx context.Context, classes []*model.Class) (int, error) {
	if len(classes) < 20 {
		return 0, fmt.Errorf("store: more classes expected for update")
	}

	for _, c := range classes {
		if !model.IsValidClassNumber(c.Number) {
			return 0, fmt.Errorf("invalid class number %d", c.Number)
		}
	}

	var mutationCount int

	_, err := store.dsClient.RunInTransaction(ctx, func(tx *datastore.Transaction) error {

		xhashes := make(map[int]string)

		// Step 1: Query for import field hash values.

		var hashValues []classΠImportHash
		keys, err := store.dsClient.GetAll(ctx,
			datastore.NewQuery(classKind).Ancestor(conferenceEntityGroupKey).Project(model.Class_ImportHash),
			&hashValues)
		if err != nil {
			return err
		}

		for i, k := range keys {
			xhashes[int(k.ID)] = hashValues[i].ImportHash
		}

		// Step 2: For each class insert or update...

		var mutations []*datastore.Mutation

		for _, c := range classes {
			key := classKey(c.Number)
			hash := c.HashImportFields()
			xhash, ok := xhashes[c.Number]
			if !ok {
				// New class.
				c.ImportHash = hash
				mutations = append(mutations, datastore.NewInsert(classKey(c.Number), (*xClass)(c)))
				continue
			}
			delete(xhashes, c.Number)
			if hash == xhash {
				continue
			}
			// Modified class.
			var xc xClass
			if err := tx.Get(key, &xc); err != nil {
				return err
			}
			xc.ImportHash = hash
			c.CopyImportFieldsTo((*model.Class)(&xc))
			mutations = append(mutations, datastore.NewUpdate(key, &xc))
		}

		// Step 3: Delete classes missing from the imported data.

		for number := range xhashes {
			mutations = append(mutations, datastore.NewDelete(classKey(number)))
		}

		mutationCount = len(mutations)
		if mutationCount == 0 {
			return nil
		}

		_, err = tx.Mutate(mutations...)
		return err
	})

	return mutationCount, err
}

// UpdateParticipants gets and puts all entities. Use when adding new indexed fields to the entity.
func (store *Store) UpdateClasses(ctx context.Context) error {
	keys, err := store.dsClient.GetAll(ctx, datastore.NewQuery(classKind).Ancestor(conferenceEntityGroupKey).KeysOnly(), nil)
	if err != nil {
		return err
	}
	_, err = store.updateEntities(ctx, keys, func(xc *xClass) error { return nil })
	return err
}
