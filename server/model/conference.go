package model

import (
	"strings"
	"sync"
)

//go:generate go run gogen.go -input conference.go -output gen_conference.go Conference

type Lunch struct {
	Name      string `json:"name" datastore:"name,noindex"`
	ShortName string `json:"shortName" datastore:"shortName,noindex"`
	Location  string `json:"location" datastore:"location,noindex"`

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

	RegistrationURL string `json:"registrationURL" datastore:"registrationURL,noindex,omitempty"`

	// Use this message to announce when registration will open or that the
	// current catalog is for the previous event.
	CatalogStatusMessage string `json:"catalogStatusMessage" datastore:"catalogStatusMessage,noindex,omitempty"`

	NoClassDescription   string `json:"noClassDescription" datastore:"noClassDescription,noindex,omitempty"`
	OABanquetDescription string `json:"oaBanquetDescription" datastore:"oaBanquetDescription,noindex,omitempty"`

	// Whitespace separated email addresses.
	StaffIDs string `json:"staffIDs" datastore:"staffIDs,noindex,omitempty"`

	OABanquetLocation string `json:"oaBanquetLocation" datastore:"oaBanquetLocation,noindex"`
	OpeningLocation   string `json:"openingLocation" datastore:"openingLocation,noindex"`

	once     sync.Once
	staffMap map[string]bool
	lunch    struct {
		def        *Lunch
		byClass    map[int]*Lunch
		byUnitType map[string]*Lunch
	}
}

func (c *Conference) IsStaff(id string) bool {
	c.setup()
	return c.staffMap[id]
}

func (c *Conference) setup() {
	c.once.Do(func() {
		c.staffMap = make(map[string]bool)
		for _, id := range strings.Fields(c.StaffIDs) {
			c.staffMap[strings.ToLower(id)] = true
		}
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

var (
	tbdLunch     = &Lunch{Seating: 2, Name: "TBD", ShortName: "TBD", Location: "TBD"}
	programLunch = &Lunch{Seating: 2, Name: "Lunch location depends on participant unit type", ShortName: "*"}
)

func (c *Conference) ClassLunch(class *Class) *Lunch {
	c.setup()
	start, end := class.StartEnd()
	if start > 2 || end < 2 {
		return nil
	}
	l := c.lunch.byClass[class.Number]
	if l == nil {
		l = programLunch
	}
	return l
}

func (c *Conference) ParticipantLunch(p *Participant) *Lunch {
	c.setup()
	var skipClasses bool
	for _, ic := range p.InstructorClasses {
		if ic.Session == LunchSession {
			if l, ok := c.lunch.byClass[ic.Class]; ok {
				return l
			}
			skipClasses = true
			break
		}
	}
	if !skipClasses {
		for _, class := range p.Classes {
			if l, ok := c.lunch.byClass[class]; ok {
				return l
			}
		}
	}
	if l, ok := c.lunch.byUnitType[p.UnitType]; ok {
		return l
	}
	if len(c.Lunches) == 0 {
		return tbdLunch
	}
	return c.Lunches[0]
}
