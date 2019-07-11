package store

import (
	"context"
	"sort"

	"github.com/seaptc/server/model"
	"golang.org/x/sync/errgroup"

	"cloud.google.com/go/datastore"
)

const (
	classEvaluationKind      = "classEvaluation"
	conferenceEvaluationKind = "confEvaluation"
)

// classEvaluationΠClass is used as the destination type for project(class) queries.
type classEvaluationΠClass struct {
	Class int `datastore:"class"`
}

type xClassEvaluation model.ClassEvaluation
type xConferenceEvaluation model.ConferenceEvaluation

func (e *xClassEvaluation) model() *model.ClassEvaluation {
	return (*model.ClassEvaluation)(e)
}

func (e *xConferenceEvaluation) model() *model.ConferenceEvaluation {
	return (*model.ConferenceEvaluation)(e)
}

func (e *xClassEvaluation) LoadKey(k *datastore.Key) error {
	e.Session = int(k.ID - 1)
	e.ParticipantID = k.Parent.Name
	return nil
}

func (e *xConferenceEvaluation) LoadKey(k *datastore.Key) error {
	e.ParticipantID = k.Parent.Name
	return nil
}

func classEvaluationKey(participantID string, session int) *datastore.Key {
	return datastore.IDKey(classEvaluationKind, int64(session)+1, participantKey(participantID))
}

func conferenceEvaluationKey(participantID string) *datastore.Key {
	return datastore.IDKey(conferenceEvaluationKind, 1, participantKey(participantID))
}

func (store *Store) GetClassEvaluation(ctx context.Context, participantID string, session int) (*model.ClassEvaluation, error) {
	var xe xClassEvaluation
	err := store.dsClient.Get(ctx, classEvaluationKey(participantID, session), &xe)
	return xe.model(), err
}

func (store *Store) GetConferenceEvaluation(ctx context.Context, participantID string) (*model.ConferenceEvaluation, error) {
	var xe xConferenceEvaluation
	err := store.dsClient.Get(ctx, conferenceEvaluationKey(participantID), &xe)
	return xe.model(), err
}

func (store *Store) SetClassEvaluation(ctx context.Context, eval *model.ClassEvaluation) error {
	_, err := store.dsClient.Put(ctx, classEvaluationKey(eval.ParticipantID, eval.Session), (*xClassEvaluation)(eval))
	return err
}

func (store *Store) SetConferenceEvaluation(ctx context.Context, eval *model.ConferenceEvaluation) error {
	_, err := store.dsClient.Put(ctx, conferenceEvaluationKey(eval.ParticipantID), (*xConferenceEvaluation)(eval))
	return err
}

func (store *Store) GetRecordedEvaluations(ctx context.Context, participantID string, cms *model.ClassMaps) (bool, []*model.SessionClass, error) {
	var (
		g          errgroup.Group
		conference bool
	)

	g.Go(func() error {
		_, err := store.GetConferenceEvaluation(ctx, participantID)
		switch {
		case err == nil:
			conference = true
		case err != ErrNotFound:
			return err
		}
		return nil
	})

	var classes []classEvaluationΠClass
	query := datastore.NewQuery(classEvaluationKind).
		Ancestor(participantKey(participantID)).
		Project(model.ClassEvaluation_Class)
	keys, err := store.dsClient.GetAll(ctx, query, &classes)
	if err != nil {
		return false, nil, err
	}
	var sessionClasses []*model.SessionClass
	for i := range classes {
		c := cms.ClassByNumber[classes[i].Class]
		if c == nil {
			// TODO handle missing class
			continue
		}
		session := int(keys[i].ID) - 1
		sessionClasses = append(sessionClasses, &model.SessionClass{
			Class:   c,
			Session: session,
			Part:    session - c.Start() + 1,
		})
	}

	sort.Slice(sessionClasses, func(i, j int) bool {
		return sessionClasses[i].Session < sessionClasses[j].Session
	})

	if err := g.Wait(); err != nil {
		return false, nil, err
	}

	return conference, sessionClasses, nil
}
