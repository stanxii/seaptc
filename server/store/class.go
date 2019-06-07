package store

import (
	"context"
	"fmt"
	"log"

	"github.com/seaptc/server/model"

	"cloud.google.com/go/datastore"
)

const classKind = "class"

var classesEntityGroupKey = datastore.IDKey("classkind", 1, nil)

func classKey(number int) *datastore.Key {
	return datastore.IDKey(classKind, int64(number), classesEntityGroupKey)
}

var deletedClassFields = map[string]bool{
	"dkNeedsUpdate": true,
	"titleNotes":    true,
}

// xClass overrides datastore load and save on an model.Class.
type xClass model.Class

func (c *xClass) Load(ps []datastore.Property) error {
	return datastore.LoadStruct((*model.Class)(c), filterProperties(ps, deletedClassFields))
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

var (
	allClassesQuery = datastore.NewQuery(classKind).Ancestor(classesEntityGroupKey).Project(
		model.Class_Number,
		model.Class_Length,
		model.Class_Title,
		model.Class_Capacity,
		model.Class_Location,
		model.Class_Responsibility)
	allClassesFullQuery = datastore.NewQuery(classKind).Ancestor(classesEntityGroupKey)
)

func (store *Store) getAllClasses(ctx context.Context, q *datastore.Query) ([]*model.Class, error) {
	var xclasses []*xClass
	_, err := store.dsClient.GetAll(ctx, q, &xclasses)
	if err != nil {
		log.Println("BOOOM")
		return nil, err
	}
	classes := make([]*model.Class, len(xclasses))
	for i, xc := range xclasses {
		classes[i] = (*model.Class)(xc)
	}
	return classes, nil
}

func (store *Store) GetAllClasses(ctx context.Context) ([]*model.Class, error) {
	return store.getAllClasses(ctx, allClassesQuery)
}

func (store *Store) GetAllClassesFull(ctx context.Context) ([]*model.Class, error) {
	return store.getAllClasses(ctx, allClassesFullQuery)
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

		// Step 1: Get all keys

		keys, err := store.dsClient.GetAll(ctx,
			datastore.NewQuery(classKind).Ancestor(classesEntityGroupKey).KeysOnly(), nil)
		if err != nil {
			return err
		}

		for _, k := range keys {
			xhashes[int(k.ID)] = ""
		}

		// Step 2: Query for import field hash values.

		var hashValues []struct {
			Hash string `datastore:"importHash"`
		}

		keys, err = store.dsClient.GetAll(ctx,
			datastore.NewQuery(classKind).Ancestor(classesEntityGroupKey).Project(model.Class_ImportHash),
			&hashValues)
		if err != nil {
			return err
		}

		for i, k := range keys {
			xhashes[int(k.ID)] = hashValues[i].Hash
		}

		// Step 3: For each particpant either insert or update...

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

		// Step 4: Delete classes missing from the imported data.

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
