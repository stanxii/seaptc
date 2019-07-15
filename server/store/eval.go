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

type xSessionEvaluation model.SessionEvaluation
type xConferenceEvaluation model.ConferenceEvaluation

func (e *xSessionEvaluation) model() *model.SessionEvaluation {
	return (*model.SessionEvaluation)(e)
}

func (e *xConferenceEvaluation) model() *model.ConferenceEvaluation {
	return (*model.ConferenceEvaluation)(e)
}

func (e *xSessionEvaluation) Load(ps []datastore.Property) error {
	return datastore.LoadStruct(e.model(), ps)
}

func (e *xSessionEvaluation) LoadKey(k *datastore.Key) error {
	e.Session = int(k.ID - 1)
	if k := k.Parent; k != nil && k.Kind == participantKind {
		e.ParticipantID = k.Name
	}
	return nil
}

func (e *xSessionEvaluation) Save() ([]datastore.Property, error) {
	return datastore.SaveStruct(e.model())
}

func (e *xConferenceEvaluation) Load(ps []datastore.Property) error {
	return datastore.LoadStruct(e.model(), ps)
}

func (e *xConferenceEvaluation) LoadKey(k *datastore.Key) error {
	if k := k.Parent; k != nil && k.Kind == participantKind {
		e.ParticipantID = k.Name
	}
	return nil
}

func (e *xConferenceEvaluation) Save() ([]datastore.Property, error) {
	return datastore.SaveStruct(e.model())
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
	var xeval xSessionEvaluation
	err := store.dsClient.Get(ctx, sessionEvaluationKey(participantID, session), &xeval)
	return xeval.model(), err
}

func (store *Store) GetSessionEvaluations(ctx context.Context, participantID string) ([]*model.SessionEvaluation, error) {
	if participantID == "" {
		return nil, errInvalidParticipantID
	}
	var xevals []*xSessionEvaluation
	query := datastore.NewQuery(sessionEvaluationKind).Ancestor(participantKey(participantID))
	_, err := store.dsClient.GetAll(ctx, query, &xevals)
	if err != nil {
		return nil, err
	}
	evals := make([]*model.SessionEvaluation, len(xevals))
	for i := range xevals {
		evals[i] = xevals[i].model()
	}
	return evals, nil
}

func (store *Store) SetSessionEvaluations(ctx context.Context, sessionEvaluations []*model.SessionEvaluation) error {
	keys := make([]*datastore.Key, len(sessionEvaluations))
	xevals := make([]*xSessionEvaluation, len(sessionEvaluations))
	for i, eval := range sessionEvaluations {
		if eval.ParticipantID == "" {
			return errInvalidParticipantID
		}
		keys[i] = sessionEvaluationKey(eval.ParticipantID, eval.Session)
		xevals[i] = (*xSessionEvaluation)(eval)
	}
	_, err := store.dsClient.PutMulti(ctx, keys, xevals)
	return err
}

func (store *Store) GetConferenceEvaluation(ctx context.Context, participantID string) (*model.ConferenceEvaluation, error) {
	if participantID == "" {
		return nil, errInvalidParticipantID
	}
	var xeval xConferenceEvaluation
	err := store.dsClient.Get(ctx, conferenceEvaluationKey(participantID), &xeval)
	return xeval.model(), err
}

func (store *Store) SetConferenceEvaluation(ctx context.Context, conferenceEvaluation *model.ConferenceEvaluation) error {
	if conferenceEvaluation.ParticipantID == "" {
		return errInvalidParticipantID
	}
	key := conferenceEvaluationKey(conferenceEvaluation.ParticipantID)
	_, err := store.dsClient.Put(ctx, key, (*xConferenceEvaluation)(conferenceEvaluation))
	return err
}

func (store *Store) GetAllConferenceEvaluations(ctx context.Context) ([]*model.ConferenceEvaluation, error) {
	query := datastore.NewQuery(conferenceEvaluationKind).Ancestor(conferenceEntityGroupKey)
	var evals []*xConferenceEvaluation
	_, err := store.dsClient.GetAll(ctx, query, &evals)
	if err != nil {
		return nil, err
	}
	conferenceEvaluations := make([]*model.ConferenceEvaluation, 0, len(evals))
	for _, eval := range evals {
		if eval.ParticipantID == "" {
			continue
		}
		conferenceEvaluations = append(conferenceEvaluations, eval.model())
	}
	return conferenceEvaluations, nil
}

func (store *Store) GetAllSessionEvaluations(ctx context.Context) ([]*model.SessionEvaluation, error) {
	query := datastore.NewQuery(sessionEvaluationKind).Ancestor(conferenceEntityGroupKey)
	var evals []*xSessionEvaluation
	_, err := store.dsClient.GetAll(ctx, query, &evals)
	if err != nil {
		return nil, err
	}
	sessionEvaluations := make([]*model.SessionEvaluation, 0, len(evals))
	for _, eval := range evals {
		if eval.ParticipantID == "" {
			continue
		}
		sessionEvaluations = append(sessionEvaluations, eval.model())
	}
	return sessionEvaluations, nil
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
