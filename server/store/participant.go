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

func (store *Store) GetParticipant(ctx context.Context, id string) (*model.Participant, error) {
	var p model.Participant
	err := store.dsClient.Get(ctx, participantKey(id), &p)
	return &p, err
}

func (store *Store) GetParticipantForLoginCode(ctx context.Context, loginCode string) (*model.Participant, error) {
	if loginCode == "" {
		return nil, ErrNotFound
	}
	var participants []*model.Participant
	_, err := store.dsClient.GetAll(ctx, datastore.NewQuery(participantKind).
		Ancestor(conferenceEntityGroupKey).
		Filter(model.Participant_LoginCode+"=", loginCode), &participants)
	if err != nil {
		return nil, err
	}
	if len(participants) != 1 {
		return nil, ErrNotFound
	}
	return participants[0], nil
}

func (store *Store) GetParticipantsByID(ctx context.Context, ids []string) ([]*model.Participant, error) {
	keys := make([]*datastore.Key, len(ids))
	for i, id := range ids {
		keys[i] = participantKey(id)
	}
	participants := make([]*model.Participant, len(ids))
	err := noEntityOK(store.dsClient.GetMulti(ctx, keys, participants))
	if err != nil {
		return nil, err
	}
	j := 0
	for _, p := range participants {
		if p != nil {
			participants[j] = p
			j++
		}
	}
	return participants[:j], nil
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
		g            errgroup.Group
		participants []*model.Participant
		keys         []*datastore.Key
		classes      []participantΠClass
		ikeys        []*datastore.Key
		iclasses     []participantΠInstructorClass
	)

	g.Go(func() error {
		var err error
		_, err = store.dsClient.GetAll(ctx, allParticipantsQuery, &participants)
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

	for _, p := range participants {
		p.Classes = cmap[p.ID]
		sort.Ints(p.Classes)

		p.InstructorClasses = icmap[p.ID]
		model.SortInstructorClasses(p.InstructorClasses)
	}
	return participants, nil
}

func (store *Store) GetAllParticipantsFull(ctx context.Context) ([]*model.Participant, error) {
	var participants []*model.Participant
	_, err := store.dsClient.GetAll(ctx, datastore.NewQuery(participantKind).Ancestor(conferenceEntityGroupKey), &participants)
	return participants, err
}

func (store *Store) GetClassParticipants(ctx context.Context, classNumber int) ([]*model.Participant, error) {
	keys, err := store.dsClient.GetAll(ctx, datastore.NewQuery(participantKind).
		Ancestor(conferenceEntityGroupKey).
		Filter(model.Participant_Classes+"=", classNumber).
		KeysOnly(), nil)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(keys))
	for i, key := range keys {
		ids[i] = key.Name
	}
	return store.GetParticipantsByID(ctx, ids)
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

	hashes := make(map[string]string)
	for _, p := range participants {
		hashes[participantID(p)] = p.HashImportFields()
	}

	var allAdds, allUpdates []string
	var xhashes map[string]string

	for len(participants) > 0 {

		var adds, updates []string
		var offset int

		_, err := store.dsClient.RunInTransaction(ctx, func(tx *datastore.Transaction) error {

			adds = adds[:0]
			updates = updates[:0]

			// Query for import field hash values and login codes

			xhashes = make(map[string]string)
			codes := make(map[string]bool)
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

			// For each participanti, insert or update as needed...

			var mutations []*datastore.Mutation

			for offset = 0; offset < len(participants) && len(mutations) < maxMutationsPerCall; offset++ {
				p := participants[offset]
				id := participantID(p)
				hash := hashes[id]
				xhash := xhashes[id]
				if hash == xhash {
					// No change to participant, continue to next.
					continue
				}

				key := participantKey(id)
				if xhash == "" {
					// Participant not in datastore, insert.
					p.ImportHash = hash
					p.PrintForm = true
					p.LoginCode, err = allocateUniqueLoginCode(codes)
					if err != nil {
						return err
					}
					mutations = append(mutations, datastore.NewInsert(key, p))
					adds = append(adds, p.LastName)
					continue
				} else {
					// Participant is in datastore, update.
					var xp model.Participant
					if err := tx.Get(key, &xp); err != nil {
						return err
					}
					xp.ImportHash = hash
					xp.PrintForm = xp.PrintForm || !p.EqualPrintFields(&xp)
					p.CopyImportFieldsTo(&xp)
					mutations = append(mutations, datastore.NewUpdate(key, &xp))
					updates = append(updates, p.LastName)
				}
			}

			_, err = tx.Mutate(mutations...)
			return err
		})

		if err != nil {
			return "", err
		}

		participants = participants[offset:]
		allAdds = append(allAdds, adds...)
		allUpdates = append(allUpdates, updates...)
	}

	// Find particpants to delete.

	for id := range hashes {
		delete(xhashes, id)
	}

	/*
		const deleteLimit = 20
		if len(xhashes) > deleteLimit {
			return "", fmt.Errorf("possible bad import, attempt to delete %d participants, limit is %d", len(xhashes), deleteLimit)
		}
	*/

	for id := range xhashes {
		if err := noEntityOK(store.dsClient.Delete(ctx, participantKey(id))); err != nil {
			return "", err
		}
	}

	// Create summary of the change.
	var parts []string
	if len(allAdds) > 0 {
		parts = append(parts, fmt.Sprintf("Added %s", joinComma(allAdds, 5)))
	}
	if len(allUpdates) > 0 {
		parts = append(parts, fmt.Sprintf("Updated %s", joinComma(allUpdates, 5)))
	}
	if len(xhashes) > 0 {
		parts = append(parts, fmt.Sprintf("Deleted %d", len(xhashes)))
	}
	summary := strings.Join(parts, "; ")

	return summary, nil
}

func equalInstructorClasses(a []model.InstructorClass, b []model.InstructorClass) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (store *Store) SetInstructorClasses(ctx context.Context, participantID string, classes []model.InstructorClass) error {
	model.SortInstructorClasses(classes)
	key := participantKey(participantID)
	return store.updateEntity(ctx, key, func(xp *model.Participant) error {
		xp.PrintForm = xp.PrintForm || !equalInstructorClasses(classes, xp.InstructorClasses)
		xp.InstructorClasses = classes
		return nil
	})
}

func (store *Store) SetNotesNoShow(ctx context.Context, participantID, notes string, noShow bool) error {
	key := participantKey(participantID)
	return store.updateEntity(ctx, key, func(xp *model.Participant) error {
		xp.Notes = notes
		xp.NoShow = noShow
		return nil
	})
}

func (store *Store) SetParticipantsPrintForm(ctx context.Context, participantIDs []string, printForm bool) (int, error) {
	keys := make([]*datastore.Key, len(participantIDs))
	for i, id := range participantIDs {
		keys[i] = participantKey(id)
	}
	return store.updateEntities(ctx, keys, func(xp *model.Participant) error {
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
	_, err = store.updateEntities(ctx, keys, func(*model.Participant) error { return nil })
	return err
}

// DebugSetParticipant overwrites participant with the given value. Use for
// debugging and testing only because can clobber other edits to the
// participant.
func (store *Store) DebugSetParticipant(ctx context.Context, p *model.Participant) error {
	key := participantKey(p.ID)
	_, err := store.dsClient.Put(ctx, key, p)
	return err
}
