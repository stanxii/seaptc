package store

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/seaptc/server/model"

	"cloud.google.com/go/datastore"
)

const participantKind = "participant"

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
	if p.Youth {
		buf = append(buf, 0, 1)
	}
	sum := md5.Sum(bytes.ToLower(buf))
	return hex.EncodeToString(sum[:])
}

func participantKey(id string) *datastore.Key {
	return datastore.NameKey(participantKind, id, conferenceEntityGroupKey)
}

// participantΠClass is used as the destination type for project(class)
// queries.
type participantΠClass struct {
	// Array proparties are returned as single elements in project queries.
	Class int `datastore:"classes"`
}

// participantπImportHashLoginCode is as destination type for project(import
// hash, login code) queries.
type participantΠImportHashLoginCode struct {
	ImportHash string `datastore:"importHash"`
	LoginCode  string `datastore:"loginCode"`
}

// xParticipant overrides datastore load and save on a model.Participant
type xParticipant model.Participant

var deletedParticipantFields = map[string]bool{}

func (p *xParticipant) Load(ps []datastore.Property) error {
	err := datastore.LoadStruct((*model.Participant)(p), filterProperties(ps, deletedParticipantFields))
	if err != nil {
		return err
	}
	(*model.Participant)(p).Init()
	return nil
}

func (p *xParticipant) LoadKey(k *datastore.Key) error {
	p.ID = k.Name
	return nil
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

func (store *Store) getParticpantClasses(ctx context.Context) ([]*datastore.Key, []participantΠClass, error) {
	var classes []participantΠClass
	// no ancestor in query for use of built-in index.
	query := datastore.NewQuery(participantKind).Project(model.Participant_Classes)
	keys, err := store.dsClient.GetAll(ctx, query, &classes)
	return keys, classes, err
}

func (store *Store) GetClassParticipantCounts(ctx context.Context) (map[int]int, error) {
	_, classes, err := store.getParticpantClasses(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[int]int)
	for _, c := range classes {
		result[c.Class]++
	}
	return result, nil
}

var allParticipantsQuery = datastore.NewQuery(participantKind).Ancestor(conferenceEntityGroupKey).Project(
	model.Participant_LastName,
	model.Participant_FirstName,
	model.Participant_Suffix,
	model.Participant_Council,
	model.Participant_District,
	model.Participant_UnitNumber,
	model.Participant_UnitType,
	model.Participant_Staff,
	model.Participant_StaffRole,
	model.Participant_Youth)

func (store *Store) GetAllParticipants(ctx context.Context) ([]*model.Participant, error) {

	// Use two project queries to get core participant fields in two read operations.

	var xparticipants []*xParticipant
	_, err := store.dsClient.GetAll(ctx, allParticipantsQuery, &xparticipants)
	if err != nil {
		return nil, err
	}

	keys, classes, err := store.getParticpantClasses(ctx)
	if err != nil {
		return nil, err
	}
	cmap := make(map[string][]int)
	for i, key := range keys {
		id := key.Name
		cmap[id] = append(cmap[id], classes[i].Class)
	}

	participants := make([]*model.Participant, len(xparticipants))
	for i, xp := range xparticipants {
		p := (*model.Participant)(xp)
		p.Classes = cmap[p.ID]
		participants[i] = p
	}
	return participants, nil
}

func (store *Store) GetClassParticipants(ctx context.Context, classNumber int) ([]*model.Participant, error) {
	keys, err := store.dsClient.GetAll(ctx, datastore.NewQuery(participantKind).
		Ancestor(conferenceEntityGroupKey).
		Filter(model.Participant_Classes+"=", classNumber).
		KeysOnly(), nil)
	if err != nil {
		return nil, err
	}

	xparticipants := make([]*xParticipant, len(keys))
	err = noEntityOK(store.dsClient.GetMulti(ctx, keys, xparticipants))
	if err != nil {
		return nil, err
	}

	participants := make([]*model.Participant, len(xparticipants))
	i := 0
	for _, xp := range xparticipants {
		if xp.ID == "" {
			// skip not found
			continue
		}
		participants[i] = (*model.Participant)(xp)
		i++
	}
	return participants[:i], nil
}

func getUniqueLoginCode(codes map[string]bool) (string, error) {
	var b [4]byte
	for i := 0; i < 10000; i++ {
		if _, err := rand.Read(b[:]); err != nil {
			return "", err
		}
		n := int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
		code := strconv.Itoa(n%899999 + 100000)
		if codes[code] {
			continue
		}
		codes[code] = true
		return code, nil
	}
	return "", errors.New("could not assign login code")
}

func (store *Store) ImportParticipants(ctx context.Context, participants []*model.Participant) (int, error) {

	var mutationCount int

	_, err := store.dsClient.RunInTransaction(ctx, func(tx *datastore.Transaction) error {

		xhashes := make(map[string]string)
		codes := make(map[string]bool)

		// Step 1: Query for import field hash values and login codes

		var hashCodeValues []participantΠImportHashLoginCode
		keys, err := store.dsClient.GetAll(ctx,
			datastore.NewQuery(participantKind).Ancestor(conferenceEntityGroupKey).Project(model.Participant_ImportHash, model.Participant_LoginCode),
			&hashCodeValues)
		if err != nil {
			return err
		}

		for i, k := range keys {
			xhashes[k.Name] = hashCodeValues[i].ImportHash
			codes[hashCodeValues[i].LoginCode] = true
		}

		// Step 3: For each participant either insert or update...

		var mutations []*datastore.Mutation

		for _, p := range participants {
			id := participantID(p)
			key := participantKey(id)
			hash := p.HashImportFields()
			xhash, ok := xhashes[id]

			if !ok {
				// Participant not in datastore, insert.
				p.ImportHash = hash
				p.LoginCode, err = getUniqueLoginCode(codes)
				if err != nil {
					return err
				}
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

		const deleteLimit = 20
		if len(xhashes) > deleteLimit {
			return fmt.Errorf("possible bad import, attempt to delete %d participants, limit is %d", len(xhashes), deleteLimit)
		}

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
