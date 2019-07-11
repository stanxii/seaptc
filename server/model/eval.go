package model

import "time"

//go:generate go run gogen.go -input eval.go -output gen_eval.go ClassEvaluation ConferenceEvaluation

type ClassEvaluation struct {
	ParticipantID      string    `json:"participantID" datastore:"-"`
	Session            int       `json:"session" datastore:"-"`
	Class              int       `json:"class" datastore:"class"`
	KnowledgeRating    int       `json:"knowledge" datastore:"knowledge,noindex"`
	PresentationRating int       `json:"promotion" datastore:"promotion,noindex"`
	UsefulnessRating   int       `json:"usefulness" datastore:"usefulness,noindex"`
	OverallRating      int       `json:"overall" datastore:"overall,noindex"`
	Comments           string    `json:"comments" datastore:"comments,noindex"`
	Source             string    `json:"source" datastore:"source,noindex"`
	Updated            time.Time `json:"updated" datastore:"updated,noindex"`
}

type ConferenceEvaluation struct {
	ParticipantID           string    `json:"participantID" datastore:"-"`
	ExperienceRating        int       `json:"experience" datastore:"experience,noindex"`
	PromotionRating         int       `json:"promotion" datastore:"promotion,noindex"`
	RegistrationRating      int       `json:"registration" datastore:"registration,noindex"`
	CheckinRating           int       `json:"checkin" datastore:"checkin,noindex"`
	MidwayRating            int       `json:"midway" datastore:"midway,noindex"`
	LunchRating             int       `json:"lunch" datastore:"lunch,noindex"`
	FacilitiesRating        int       `json:"facilities" datastore:"facilities,noindex"`
	WebsiteRating           int       `json:"website" datastore:"website,noindex"`
	SignageWayfindingRating int       `json:"signageWayfinding" datastore:"signageWayfinding,noindex"`
	LearnTopics             string    `json:"learnTopics" datastore:"learnTopics,noindex"`
	TeachTopics             string    `json:"teachTopics" datastore:"teachTopics,noindex"`
	Comments                string    `json:"comments" datastore:"comments,noindex"`
	Source                  string    `json:"source" datastore:"source,noindex"`
	Updated                 time.Time `json:"updated" datastore:"updated,noindex"`
}
