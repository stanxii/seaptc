package data

import (
	"sync"
	"time"
)

type Lunch struct {
	Name     string `json:"name" firestore:"name"`
	Location string `json:"location" firestore:"location"`

	// 1: first, 2: second
	Seating int `json:"seating" firestore:"seating"`

	// If participant is taking one of these classes then
	//  pick up lunch here
	// else if particpant is in one of these unit types then
	//  pick up lunch here
	// else
	//  pick up lunch at general
	Clases    []int    `json:"classes" firestore:"classes"`
	UnitTypes []string `json:"unitTypes" firestore:"unitTypes"`
}

type Conference struct {
	// First lunch is default choice
	Lunches []*Lunch  `json:"lunches" firestore:"lunches"`
	Date    time.Time `json:"date" firstore:"date"`

	lunch struct {
		once       sync.Once
		def        *Lunch
		byClass    map[int]*Lunch
		byUnitType map[string]*Lunch
	}
}

func (c *Conference) LookupLunch(p *Participant) *Lunch {
	c.lunch.once.Do(func() {
		c.lunch.byClass = make(map[int]*Lunch)
		c.lunch.byUnitType = make(map[string]*Lunch)
		for _, l := range c.Lunches {
			for _, n := range p.Classes {
				c.lunch.byClass[n] = l
			}
			for _, unitType := range l.UnitTypes {
				c.lunch.byUnitType[unitType] = l
			}
		}
	})
	for _, class := range p.Classes {
		if l, ok := c.lunch.byClass[class]; ok {
			return l
		}
	}
	if l, ok := c.lunch.byUnitType[p.UnitType]; ok {
		return l
	}
	return c.Lunches[0]
}
