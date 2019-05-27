package store

import (
	"context"
	"fmt"

	"github.com/seaptc/server/model"

	"cloud.google.com/go/datastore"
)

const classKind = "class"

var classesEntityGroupKey = datastore.IDKey("classkind", 1, nil)

func classKey(number int) *datastore.Key {
	return datastore.IDKey(classKind, int64(number), classesEntityGroupKey)
}

type dsClass model.Class

func (c *dsClass) Load(ps []datastore.Property) error {
	return datastore.LoadStruct((*model.Class)(c), ps)
}

func (c *dsClass) Save() ([]datastore.Property, error) {
	ps, err := datastore.SaveStruct((*model.Class)(c))
	return ps, err
}

func (store *Store) GetClass(ctx context.Context, number int) (*model.Class, error) {
	var c dsClass
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
	// XXX var classes []*dsClass
	var classes []*model.Class
	_, err := store.dsClient.GetAll(ctx, q, &classes)
	if err != nil {
		return nil, err
	}
	result := make([]*model.Class, len(classes))
	for i, c := range classes {
		result[i] = (*model.Class)(c)
	}
	return result, nil
}

func (store *Store) GetAllClasses(ctx context.Context) ([]*model.Class, error) {
	return store.getAllClasses(ctx, allClassesQuery)
}

func (store *Store) GetAllClassesFull(ctx context.Context) ([]*model.Class, error) {
	return store.getAllClasses(ctx, allClassesFullQuery)
}

func (store *Store) UpdateClassesFromSheet(ctx context.Context, classes []*model.Class, updateAll bool) (int, error) {
	if len(classes) < 20 {
		return 0, fmt.Errorf("store: more classes expected for update")
	}

	var mutationCount int

	_, err := store.dsClient.RunInTransaction(ctx, func(tx *datastore.Transaction) error {

		var xclasses []*dsClass
		_, err := store.dsClient.GetAll(ctx,
			datastore.NewQuery(classKind).Ancestor(classesEntityGroupKey).Transaction(tx),
			&xclasses)
		if err != nil {
			return err
		}

		m := make(map[int]*model.Class)
		for _, xc := range xclasses {
			m[xc.Number] = (*model.Class)(xc)
		}

		var mutations []*datastore.Mutation

		for _, c := range classes {
			xc, ok := m[c.Number]
			if !ok {
				mutations = append(mutations, datastore.NewUpsert(classKey(c.Number), (*dsClass)(c)))
				continue
			}
			if updateAll || !xc.EqualSheetFields(c) {
				xc.CopySheetFields(c)
				c.Junk1 = false // XXX
				mutations = append(mutations, datastore.NewUpsert(classKey(xc.Number), (*dsClass)(xc)))
			}
			delete(m, c.Number)
		}

		for _, xc := range m {
			mutations = append(mutations, datastore.NewDelete(classKey(xc.Number)))
		}

		if len(mutations) == 0 {
			return nil
		}

		mutationCount = len(mutations)
		_, err = tx.Mutate(mutations...)
		return err
	})

	return mutationCount, err
}
