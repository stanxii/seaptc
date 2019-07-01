package store

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/seaptc/server/model"
	"golang.org/x/sync/errgroup"

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

// participantΠinstructorClass is used as the destination type for project(instructorClass)
// queries.
type participantΠInstructorClass struct {
	// Array proparties are returned as single elements in project queries.
	InstructorClass model.InstructorClass `datastore:"instructorClasses"`
}

// participantπImportHashLoginCode is as destination type for project(import
// hash, login code) queries.
type participantΠImportHashLoginCode struct {
	ImportHash string `datastore:"importHash"`
	LoginCode  string `datastore:"loginCode"`
}

// xParticipant overrides datastore load and save on a model.Participant
type xParticipant model.Participant

var deletedParticipantFields = map[string]bool{"needsPrint": true, "printSchedule": true}

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

func (store *Store) GetParticipants(ctx context.Context, ids []string) ([]*model.Participant, error) {
	keys := make([]*datastore.Key, len(ids))
	for i, id := range ids {
		keys[i] = participantKey(id)
	}
	xparticipants := make([]*xParticipant, len(ids))
	err := noEntityOK(store.dsClient.GetMulti(ctx, keys, xparticipants))
	if err != nil {
		return nil, err
	}
	var participants []*model.Participant
	for _, xp := range xparticipants {
		if xp == nil {
			continue
		}
		participants = append(participants, (*model.Participant)(xp))
	}
	return participants, err
}

func (store *Store) getParticipantClasses(ctx context.Context) ([]*datastore.Key, []participantΠClass, error) {
	var classes []participantΠClass
	// no ancestor in query for use of built-in index.
	query := datastore.NewQuery(participantKind).Project(model.Participant_Classes)
	keys, err := store.dsClient.GetAll(ctx, query, &classes)
	return keys, classes, err
}

func (store *Store) getParticipantInstructorClasses(ctx context.Context) ([]*datastore.Key, []participantΠInstructorClass, error) {
	var classes []participantΠInstructorClass
	query := datastore.NewQuery(participantKind).
		Ancestor(conferenceEntityGroupKey).
		Project(model.Participant_InstructorClasses+".class", model.Participant_InstructorClasses+".session")
	keys, err := store.dsClient.GetAll(ctx, query, &classes)
	return keys, classes, err
}

func (store *Store) GetClassParticipantCounts(ctx context.Context) (map[int]int, error) {
	_, classes, err := store.getParticipantClasses(ctx)
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
	model.Participant_Youth,
	model.Participant_PrintForm,
	model.Participant_DietaryRestrictions)

func (store *Store) GetAllParticipants(ctx context.Context) ([]*model.Participant, error) {

	// Use three project queries to get core participant fields in three read operations.

	var (
		g             errgroup.Group
		xparticipants []*xParticipant
		keys          []*datastore.Key
		classes       []participantΠClass
		ikeys         []*datastore.Key
		iclasses      []participantΠInstructorClass
	)

	g.Go(func() error {
		var err error
		_, err = store.dsClient.GetAll(ctx, allParticipantsQuery, &xparticipants)
		return err
	})

	g.Go(func() error {
		var err error
		keys, classes, err = store.getParticipantClasses(ctx)
		return err
	})

	g.Go(func() error {
		var err error
		ikeys, iclasses, err = store.getParticipantInstructorClasses(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	cmap := make(map[string][]int)
	for i, key := range keys {
		id := key.Name
		cmap[id] = append(cmap[id], classes[i].Class)
	}

	icmap := make(map[string][]model.InstructorClass)
	for i, key := range ikeys {
		id := key.Name
		icmap[id] = append(icmap[id], iclasses[i].InstructorClass)
	}

	participants := make([]*model.Participant, len(xparticipants))
	for i, xp := range xparticipants {
		p := (*model.Participant)(xp)

		p.Classes = cmap[p.ID]
		sort.Ints(p.Classes)

		p.InstructorClasses = icmap[p.ID]
		model.SortInstructorClasses(p.InstructorClasses)

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

func allocateUniqueLoginCode(codes map[string]bool) (string, error) {
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

func joinComma(p []string, max int) string {
	if len(p) <= max {
		return strings.Join(p, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(p[:max-1], ", "), len(p)-max+1)
}

func (store *Store) ImportParticipants(ctx context.Context, participants []*model.Participant) (string, error) {

	summary := "No changes"

	_, err := store.dsClient.RunInTransaction(ctx, func(tx *datastore.Transaction) error {

		var adds, updates []string

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
				p.LoginCode, err = allocateUniqueLoginCode(codes)
				p.PrintForm = true
				if err != nil {
					return err
				}
				mutations = append(mutations, datastore.NewInsert(key, (*xParticipant)(p)))
				adds = append(adds, p.LastName)
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
			if !xp.PrintForm && !p.EqualPrintFields((*model.Participant)(&xp)) {
				xp.PrintForm = true
			}
			p.CopyImportFieldsTo((*model.Participant)(&xp))
			mutations = append(mutations, datastore.NewUpdate(key, &xp))
			updates = append(updates, p.LastName)
		}

		// Step 4: Delete participants missing from the imported data.

		const deleteLimit = 20
		if len(xhashes) > deleteLimit {
			return fmt.Errorf("possible bad import, attempt to delete %d participants, limit is %d", len(xhashes), deleteLimit)
		}

		for id := range xhashes {
			mutations = append(mutations, datastore.NewDelete(participantKey(id)))
		}

		if len(mutations) == 0 {
			return nil
		}

		_, err = tx.Mutate(mutations...)
		if err != nil {
			return err
		}

		// Create summary of the change.
		var parts []string
		if len(adds) > 0 {
			parts = append(parts, fmt.Sprintf("Added %s", joinComma(adds, 5)))
		}
		if len(updates) > 0 {
			parts = append(parts, fmt.Sprintf("Updated %s", joinComma(updates, 5)))
		}
		if len(xhashes) > 0 {
			parts = append(parts, fmt.Sprintf("Deleted %d", len(xhashes)))
		}
		summary = strings.Join(parts, "; ")

		return nil
	})

	return summary, err
}

func (store *Store) SetInstructorClasses(ctx context.Context, id string, classes []model.InstructorClass) error {
	key := participantKey(id)
	return store.updateEntity(ctx, key, func(xp *xParticipant) error {
		xp.InstructorClasses = classes
		return nil
	})
}

func (store *Store) SetParticipantsPrintForm(ctx context.Context, participantIDs []string, printForm bool) (int, error) {
	keys := make([]*datastore.Key, len(participantIDs))
	for i, id := range participantIDs {
		keys[i] = participantKey(id)
	}
	return store.updateEntities(ctx, keys, func(xp *xParticipant) error {
		if xp.PrintForm == printForm {
			return errNoUpdate
		}
		xp.PrintForm = printForm
		return nil
	})
}

// UpdateParticipants gets and puts all entities. Use when adding new indexed fields to the entity.
func (store *Store) UpdateParticipants(ctx context.Context) error {
	keys, err := store.dsClient.GetAll(ctx, datastore.NewQuery(participantKind).Ancestor(conferenceEntityGroupKey).KeysOnly(), nil)
	if err != nil {
		return err
	}
	_, err = store.updateEntities(ctx, keys, func(*xParticipant) error { return nil })
	return err
}

// DebugSetParticipant overwrites participant with the given value. Use for
// debugging and testing only because can clobber other edits to the
// participant.
func (store *Store) DebugSetParticipant(ctx context.Context, p *model.Participant) error {
	key := participantKey(p.ID)
	_, err := store.dsClient.Put(ctx, key, (*xParticipant)(p))
	return err
}
