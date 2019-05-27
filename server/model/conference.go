package model

import (
	"sync"
	"time"
)

//go:generate go run gogen.go -input conference.go -output gen_conference.go Conference

type Lunch struct {
	Name     string `json:"name" datastore:"name,noindex"`
	Location string `json:"location" datastore:"location,noindex"`

	// 1: first, 2: second
	Seating int `json:"seating" datastore:"seating,noindex"`

	// If participant is taking one of these classes then
	//  pick up lunch here
	// else if participant is in one of these unit types then
	//  pick up lunch here
	// else
	//  pick up lunch at general
	//
	// Unit types are from registration: Pack, Troop, Crew, Ship
	//
	Classes   []int    `json:"classes" datastore:"classes,noindex"`
	UnitTypes []string `json:"unitTypes" datastore:"unitTypes,noindex"`
}

type Conference struct {
	// First lunch is default choice
	Lunches []*Lunch `json:"lunches" datastore:"lunches,noindex"`

	Year  int `json:"year" datastore:"year,noindex"`
	Month int `json:"month" datastore:"month,noindex"`
	Day   int `json:"day" datastore:"day,noindex"`

	RegistrationURL string `json:"registrationURL" datastore:"registrationURL,noindex"`

	lunch struct {
		once       sync.Once
		def        *Lunch
		byClass    map[int]*Lunch
		byUnitType map[string]*Lunch
	}
}

func (c *Conference) Date() time.Time {
	return time.Date(c.Year, time.Month(c.Month), c.Day, 0, 0, 0, 0, TimeLocation)
}

func (c *Conference) setupLunch() {
	c.lunch.once.Do(func() {
		c.lunch.byClass = make(map[int]*Lunch)
		c.lunch.byUnitType = make(map[string]*Lunch)
		for _, l := range c.Lunches {
			for _, n := range l.Classes {
				c.lunch.byClass[n] = l
			}
			for _, unitType := range l.UnitTypes {
				c.lunch.byUnitType[unitType] = l
			}
		}
	})
}
