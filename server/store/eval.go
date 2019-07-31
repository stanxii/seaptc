package store

import (
	"context"
	"errors"

	"github.com/seaptc/server/model"
	"golang.org/x/sync/errgroup"

	"cloud.google.com/go/datastore"
)

const (
	sessionEvaluationKind    = "sessionEvaluation"
	conferenceEvaluationKind = "confEvaluation"
)

// sessionEvaluationΠClass is used as the destination type for project(class) queries.
type sessionEvaluationΠClass struct {
	ClassNumber int `datastore:"classNumber"`
}

func sessionEvaluationKey(participantID string, session int) *datastore.Key {
	return datastore.IDKey(sessionEvaluationKind, int64(session)+1, participantKey(participantID))
}

func conferenceEvaluationKey(participantID string) *datastore.Key {
	return datastore.IDKey(conferenceEvaluationKind, 1, participantKey(participantID))
}

var errInvalidParticipantID = errors.New("invalid participant ID")

func (store *Store) GetSessionEvaluation(ctx context.Context, participantID string, session int) (*model.SessionEvaluation, error) {
	if participantID == "" {
		return nil, errInvalidParticipantID
	}
	var e model.SessionEvaluation
	err := store.dsClient.Get(ctx, sessionEvaluationKey(participantID, session), &e)
	return &e, err
}

func (store *Store) GetSessionEvaluations(ctx context.Context, participantID string) ([]*model.SessionEvaluation, error) {
	if participantID == "" {
		return nil, errInvalidParticipantID
	}
	var evals []*model.SessionEvaluation
	query := datastore.NewQuery(sessionEvaluationKind).Ancestor(participantKey(participantID))
	_, err := store.dsClient.GetAll(ctx, query, &evals)
	return evals, err
}

func (store *Store) SetSessionEvaluations(ctx context.Context, evals []*model.SessionEvaluation) error {
	keys := make([]*datastore.Key, len(evals))
	for i, e := range evals {
		if e.ParticipantID == "" {
			return errInvalidParticipantID
		}
		keys[i] = sessionEvaluationKey(e.ParticipantID, e.Session)
	}
	_, err := store.dsClient.PutMulti(ctx, keys, evals)
	return err
}

func (store *Store) GetConferenceEvaluation(ctx context.Context, participantID string) (*model.ConferenceEvaluation, error) {
	if participantID == "" {
		return nil, errInvalidParticipantID
	}
	var e model.ConferenceEvaluation
	err := store.dsClient.Get(ctx, conferenceEvaluationKey(participantID), &e)
	return &e, err
}

func (store *Store) SetConferenceEvaluation(ctx context.Context, e *model.ConferenceEvaluation) error {
	if e.ParticipantID == "" {
		return errInvalidParticipantID
	}
	key := conferenceEvaluationKey(e.ParticipantID)
	_, err := store.dsClient.Put(ctx, key, e)
	return err
}

func (store *Store) GetAllConferenceEvaluations(ctx context.Context) ([]*model.ConferenceEvaluation, error) {
	query := datastore.NewQuery(conferenceEvaluationKind).Ancestor(conferenceEntityGroupKey)
	var evals []*model.ConferenceEvaluation
	_, err := store.dsClient.GetAll(ctx, query, &evals)
	if err != nil {
		return nil, err
	}
	j := 0
	for _, e := range evals {
		if e.ParticipantID == "" {
			continue
		}
		evals[j] = e
		j++
	}
	return evals[:j], nil
}

func (store *Store) GetAllSessionEvaluations(ctx context.Context) ([]*model.SessionEvaluation, error) {
	query := datastore.NewQuery(sessionEvaluationKind).Ancestor(conferenceEntityGroupKey)
	var evals []*model.SessionEvaluation
	_, err := store.dsClient.GetAll(ctx, query, &evals)
	if err != nil {
		return nil, err
	}
	j := 0
	for _, e := range evals {
		if e.ParticipantID == "" {
			continue
		}
		evals[j] = e
		j++
	}
	return evals[:j], nil
}

type EvaluationStatus struct {
	Conference   bool
	ClassNumbers [model.NumSession]int
}

func (store *Store) GetEvaluationStatus(ctx context.Context, participantID string) (*EvaluationStatus, error) {
	if participantID == "" {
		return nil, errInvalidParticipantID
	}

	var g errgroup.Group
	var status EvaluationStatus

	g.Go(func() error {
		_, err := store.GetConferenceEvaluation(ctx, participantID)
		switch {
		case err == nil:
			status.Conference = true
		case err != ErrNotFound:
			return err
		}
		return nil
	})

	var classes []sessionEvaluationΠClass
	query := datastore.NewQuery(sessionEvaluationKind).
		Ancestor(participantKey(participantID)).
		Project(model.SessionEvaluation_ClassNumber)
	keys, err := store.dsClient.GetAll(ctx, query, &classes)
	if err != nil {
		return nil, err
	}
	for i := range classes {
		session := int(keys[i].ID) - 1
		if 0 <= session && session < model.NumSession {
			status.ClassNumbers[session] = classes[i].ClassNumber
		}
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return &status, nil
}

func (store *Store) GetAllEvaluationStatus(ctx context.Context) (map[string]*EvaluationStatus, error) {

	var g errgroup.Group
	var conferenceKeys []*datastore.Key

	g.Go(func() error {
		var err error
		query := datastore.NewQuery(conferenceEvaluationKind).Ancestor(conferenceEntityGroupKey).KeysOnly()
		conferenceKeys, err = store.dsClient.GetAll(ctx, query, nil)
		return err
	})

	var classes []sessionEvaluationΠClass
	query := datastore.NewQuery(sessionEvaluationKind).Ancestor(conferenceEntityGroupKey).Project(model.SessionEvaluation_ClassNumber)
	keys, err := store.dsClient.GetAll(ctx, query, &classes)
	if err != nil {
		return nil, err
	}

	result := map[string]*EvaluationStatus{}
	for i := range classes {
		participantID := keys[i].Parent.Name
		session := int(keys[i].ID) - 1
		if 0 <= session && session < model.NumSession {
			status := result[participantID]
			if status == nil {
				status = &EvaluationStatus{}
				result[participantID] = status
			}
			status.ClassNumbers[session] = classes[i].ClassNumber
		}
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	for _, k := range conferenceKeys {
		participantID := k.Parent.Name
		status := result[participantID]
		if status == nil {
			status = &EvaluationStatus{}
			result[participantID] = status
		}
		status.Conference = true
	}

	return result, nil
}
