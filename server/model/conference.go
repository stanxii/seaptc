package model

import (
	"sync"
	"time"
)

//go:generate go run gogen.go -input conference.go -output gen_conference.go Conference

type Lunch struct {
	Name     string `json:"name" firestore:"name"`
	Location string `json:"location" firestore:"location"`

	// 1: first, 2: second
	Seating int `json:"seating" firestore:"seating"`

	// If participant is taking one of these classes then
	//  pick up lunch here
	// else if participant is in one of these unit types then
	//  pick up lunch here
	// else
	//  pick up lunch at general
	//
	// Unit types are from registration: Pack, Troop, Crew, Ship
	//
	Clases    []int    `json:"classes" firestore:"classes"`
	UnitTypes []string `json:"unitTypes" firestore:"unitTypes"`
}

type Conference struct {
	// First lunch is default choice
	Lunches []*Lunch `json:"lunches" firestore:"lunches"`

	Year  int `json:"year" firestore:"year"`
	Month int `json:"month" firestore:"month"`
	Day   int `json:"day" firestore:"day"`

	// String with lines in the following format:
	//  code nnn,nnn!,nnn description
	// where code is a program code (cub, bsa, ven, ...), nnn is a class
	// number, nnn! is a required class number.
	SuggestedSchedules int `json:"suggestedSchedules" firestore:"suggestedSchedules"`

	LastUpdateTime time.Time `json:"lastUpdateTime" firestore:"lastUpdateTime,serverTimestamp"`

	lunch struct {
		once       sync.Once
		def        *Lunch
		byClass    map[int]*Lunch
		byUnitType map[string]*Lunch
	}
}
