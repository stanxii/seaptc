package store

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"

	"github.com/seaptc/server/model"

	"cloud.google.com/go/datastore"
)

const participantKind = "participant"

var participantsEntityGroupKey = datastore.IDKey("participantkind", 1, nil)

// participantID returns a hash of unique particpant fields.
func participantID(p *model.Participant) string {
	var buf []byte
	buf = append(buf, p.LastName...)
	buf = append(buf, 0)
	buf = append(buf, p.FirstName...)
	buf = append(buf, 0)
	buf = append(buf, p.Suffix...)
	buf = append(buf, 0)
	buf = append(buf, p.RegistrationNumber...)
	sum := md5.Sum(bytes.ToLower(buf))
	return hex.EncodeToString(sum[:])
}

func participantKey(id string) *datastore.Key {
	return datastore.NameKey(participantKind, id, participantsEntityGroupKey)
}

// xParticipant overrides datastore load and save on a model.Participant
type xParticipant model.Participant

func (p *xParticipant) Load(ps []datastore.Property) error {
	return datastore.LoadStruct((*model.Participant)(p), ps)
}

func (p *xParticipant) Save() ([]datastore.Property, error) {
	ps, err := datastore.SaveStruct((*model.Participant)(p))
	return ps, err
}

func (store *Store) GetParticipant(ctx context.Context, id string) (*model.Participant, error) {
	var xp xParticipant
	err := store.dsClient.Get(ctx, participantKey(id), &xp)
	return (*model.Participant)(&xp), err
}

var (
	allparticipantsQuery = datastore.NewQuery(participantKind).Ancestor(participantsEntityGroupKey).Project()
)

func (store *Store) getAllParticipants(ctx context.Context, q *datastore.Query) ([]*model.Participant, error) {
	var xparticipants []*xParticipant
	_, err := store.dsClient.GetAll(ctx, q, &xparticipants)
	if err != nil {
		return nil, err
	}
	participants := make([]*model.Participant, len(xparticipants))
	for i, xp := range xparticipants {
		p := (*model.Participant)(xp)
		p.ID = participantID(p)
		participants[i] = p
	}
	return participants, nil
}

func (store *Store) GetAllParticipants(ctx context.Context) ([]*model.Participant, error) {
	return store.getAllParticipants(ctx, allparticipantsQuery)
}

func (store *Store) ImportParticipants(ctx context.Context, participants []*model.Participant) (int, error) {

	var mutationCount int

	_, err := store.dsClient.RunInTransaction(ctx, func(tx *datastore.Transaction) error {

		xhashes := make(map[string]string)

		// Step 1: Get all keys

		keys, err := store.dsClient.GetAll(ctx,
			datastore.NewQuery(participantKind).Ancestor(participantsEntityGroupKey).KeysOnly(), nil)
		if err != nil {
			return err
		}

		for _, k := range keys {
			xhashes[k.Name] = ""
		}

		// Step 2: Query for import field hash values.
		var hashValues []struct {
			Hash string `datastore:"importHash"`
		}

		keys, err = store.dsClient.GetAll(ctx,
			datastore.NewQuery(participantKind).Ancestor(participantsEntityGroupKey).Project(model.Participant_ImportHash),
			&hashValues)
		if err != nil {
			return err
		}

		for i, k := range keys {
			xhashes[k.Name] = hashValues[i].Hash
		}

		// Step 3: For each particpant either insert or update...

		var mutations []*datastore.Mutation

		for _, p := range participants {
			id := participantID(p)
			key := participantKey(id)
			hash := p.HashImportFields()
			xhash, ok := xhashes[id]

			if !ok {
				// Participant not in datastore, insert.
				p.ImportHash = hash
				mutations = append(mutations, datastore.NewInsert(key, (*xParticipant)(p)))
				continue
			}
			delete(xhashes, id)
			if hash == xhash {
				continue
			}
			// Participant is in datastore, update.
			var xp xParticipant
			if err := tx.Get(key, &xp); err != nil {
				return err
			}
			xp.ImportHash = hash
			p.CopyImportFieldsTo((*model.Participant)(&xp))
			mutations = append(mutations, datastore.NewUpdate(key, &xp))
		}

		// Step 4: Delete participants missing from the imported data.
		for id := range xhashes {
			mutations = append(mutations, datastore.NewDelete(participantKey(id)))
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
