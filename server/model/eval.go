package model

import (
	"time"

	"cloud.google.com/go/datastore"
)

// MaxEvalRating is the maximum value for an evaluation rating. The rating
// values are:
//  0 - not specified,
//  1 - minimum;
//  ...
//  MaxEvalRating - maximum
const MaxEvalRating = 4

//go:generate go run gogen.go -input eval.go -output gen_eval.go SessionEvaluation ConferenceEvaluation

type SessionEvaluation struct {
	ParticipantID      string    `json:"participantID" datastore:"-"`
	Session            int       `json:"session" datastore:"-"`
	ClassNumber        int       `json:"class" datastore:"classNumber" fields:"Edit"`
	KnowledgeRating    int       `json:"knowledge" datastore:"knowledge,noindex" fields:"Edit"`
	PresentationRating int       `json:"promotion" datastore:"promotion,noindex" fields:"Edit"`
	UsefulnessRating   int       `json:"usefulness" datastore:"usefulness,noindex" fields:"Edit"`
	OverallRating      int       `json:"overall" datastore:"overall,noindex" fields:"Edit"`
	Comments           string    `json:"comments" datastore:"comments,noindex" fields:"Edit"`
	Source             string    `json:"source" datastore:"source,noindex"`
	Updated            time.Time `json:"updated" datastore:"updated,noindex"`
}

type ConferenceEvaluation struct {
	ParticipantID           string    `json:"participantID" datastore:"-"`
	ExperienceRating        int       `json:"experience" datastore:"experience,noindex"`
	PromotionRating         int       `json:"promotion" datastore:"promotion,noindex" fields:"Edit"`
	RegistrationRating      int       `json:"registration" datastore:"registration,noindex" fields:"Edit"`
	CheckinRating           int       `json:"checkin" datastore:"checkin,noindex" fields:"Edit"`
	MidwayRating            int       `json:"midway" datastore:"midway,noindex" fields:"Edit"`
	LunchRating             int       `json:"lunch" datastore:"lunch,noindex" fields:"Edit"`
	FacilitiesRating        int       `json:"facilities" datastore:"facilities,noindex" fields:"Edit"`
	WebsiteRating           int       `json:"website" datastore:"website,noindex" fields:"Edit"`
	SignageWayfindingRating int       `json:"signageWayfinding" datastore:"signageWayfinding,noindex" fields:"Edit"`
	LearnTopics             string    `json:"learnTopics" datastore:"learnTopics,noindex" fields:"Edit"`
	TeachTopics             string    `json:"teachTopics" datastore:"teachTopics,noindex" fields:"Edit"`
	Comments                string    `json:"comments" datastore:"comments,noindex" fields:"Edit"`
	Source                  string    `json:"source" datastore:"source,noindex"`
	Updated                 time.Time `json:"updated" datastore:"updated,noindex"`
}

func (e *SessionEvaluation) Load(ps []datastore.Property) error {
	return datastore.LoadStruct(e, ps)
}

func (e *SessionEvaluation) LoadKey(k *datastore.Key) error {
	e.Session = int(k.ID - 1)
	if k := k.Parent; k != nil {
		e.ParticipantID = k.Name
	}
	return nil
}

func (e *SessionEvaluation) Save() ([]datastore.Property, error) {
	return datastore.SaveStruct(e)
}

func (e *ConferenceEvaluation) Load(ps []datastore.Property) error {
	return datastore.LoadStruct(e, ps)
}

func (e *ConferenceEvaluation) LoadKey(k *datastore.Key) error {
	if k := k.Parent; k != nil {
		e.ParticipantID = k.Name
	}
	return nil
}

func (e *ConferenceEvaluation) Save() ([]datastore.Property, error) {
	return datastore.SaveStruct(e)
}
