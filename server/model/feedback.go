package model

import "time"

//go:generate go run gogen.go -input feedback.go -output gen_feedback.go ClassEvaluation ConferenceEvaluation

type ClassEvaluation struct {
	UserID             string    `json:"userID" datastore:"-"`
	Session            int       `json:"session" datastore:"-"`
	Class              int       `json:"class" datastore:"class,noindex"`
	KnowledgeRating    int       `json:"knowledge" datastore:"knowledge,noindex"`
	PresentationRating int       `json:"promotion" datastore:"promotion,noindex"`
	UsefulnessRating   int       `json:"usefulness" datastore:"usefulness,noindex"`
	OverallRating      int       `json:"overall" datastore:"overall,noindex"`
	Comments           string    `json:"comments" datastore:"comments,noindex"`
	Source             string    `json:"source" datastore:"source,noindex"`
	Updated            time.Time `json:"updated" datastore:"updated,noindex"`
}

type ConferenceEvaluation struct {
	UserID                    string    `json:"userID" datastore:"-"`
	ExperienceRating          int       `json:"experience" datastore:"experience,noindex"`
	PromotionRating           int       `json:"promotion" datastore:"promotion,noindex"`
	OnlineRegistrationRating  int       `json:"onlineRegistration" datastore:"onlineRegistration,noindex"`
	CheckinRating             int       `json:"checkin" datastore:"checkin,noindex"`
	MidwayRating              int       `json:"midway" datastore:"midway,noindex"`
	LunchRating               int       `json:"lunch" datastore:"lunch,noindex"`
	SignageWayfindingRating   int       `json:"signageWayfinding" datastore:"signageWayfinding,noindex"`
	TechnologyRating          int       `json:"technology" datastore:"technology,noindex"`
	FacilitiesLogisticsRating int       `json:"facilitiesLogistics" datastore:"facilitiesLogistics,noindex"`
	LearnTopics               string    `json:"learnTopics" datastore:"learnTopics,noindex"`
	TeachTopics               string    `json:"teachTopics" datastore:"teachTopics,noindex"`
	Comments                  string    `json:"comments" datastore:"comments,noindex"`
	Source                    string    `json:"source" datastore:"source,noindex"`
	Updated                   time.Time `json:"updated" datastore:"updated,noindex"`
}
