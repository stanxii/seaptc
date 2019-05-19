package data

import (
	"strconv"
	"sync"
)

//go:generate go run gogen.go -output fields.go

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

/*
type Session struct {
    StartHour int `json:"startHour" firstore:"startHour"`
    StartHour int `json:"startHour" firstore:"startHour"`
    StartMinute int `json:"StartMinute" firstore:"startMinute"`
    EndHour int `json:"endHour" firstore:"endHour"`
    EndMinute int `json:"endMinute" firstore:"endMinute"`
    Lunch bool `json:"lunch" firstore:"lunch"`
}
*/

const NumSession = 6

type Conference struct {
	// First lunch is default choice
	Lunches []*Lunch `json:"lunches" firestore:"lunches"`
	//Sessions []*Session  `json:"sessions" firestore:"sessions"`

	Year  int `json:"year" firestore:"year"`
	Month int `json:"month" firestore:"month"`
	Day   int `json:"day" firestore:"day"`

	lunch struct {
		once       sync.Once
		def        *Lunch
		byClass    map[int]*Lunch
		byUnitType map[string]*Lunch
	}
}

func (c *Conference) DocName() string {
	return strconv.Itoa(c.Year)
}
