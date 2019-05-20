package model

import (
	"strings"
	"time"
)

//go:generate go run gogen.go -input participant.go -output gen_participant.go Participant

type Participant struct {
	RegistrationNumber  string `json:"registrationNumber" firestore:"registrationNumber" fields:"DK"`
	FirstName           string `json:"firstName" firestore:"firstName" fields:"DK"`
	LastName            string `json:"lastName" firestore:"lastName" fields:"DK"`
	Nickname            string `json:"nickname" firestore:"nickname" fields:"DK"`
	Suffix              string `json:"suffix" firestore:"suffix" fields:"DK"`
	Staff               bool   `json:"staff" firestore:"staff" fields:"DK"`
	Youth               bool   `json:"youth" firestore:"youth" fields:"DK"`
	Phone               string `json:"phone" firestore:"phone" fields:"DK"`
	Email               string `json:"email" firestore:"email" fields:"DK"`
	Address             string `json:"address" firestore:"address" fields:"DK"`
	City                string `json:"city" firestore:"city" fields:"DK"`
	State               string `json:"state" firestore:"state" fields:"DK"`
	Zip                 string `json:"zip" firestore:"zip" fields:"DK"`
	StaffRole           string `json:"staffRole" firestore:"staffRole" fields:"DK"` // Instructor, Support, Midway
	Council             string `json:"council" firestore:"council" fields:"DK"`
	District            string `json:"district" firestore:"district" fields:"DK"`
	UnitType            string `json:"unitType" firestore:"unitType" fields:"DK"`
	UnitNumber          string `json:"unitNumber" firestore:"unitNumber" fields:"DK"`
	DietaryRestrictions string `json:"dietaryRestrictions" firestore:"dietaryRestrictions" fields:"DK"`
	Marketing           string `json:"marketing" firestore:"marketing" fields:"DK"`
	ScoutingYears       string `json:"scoutingYears" firestore:"scoutingYears" fields:"DK"`
	ShowQRCode          bool   `json:"showQRCode" firestore:"showQRCode" fields:"DK"`
	BSANumber           string `json:"bsaNumber" firestore:"bsaNumber" fields:"DK"`
	Classes             []int  `json:"classes" firestore:"classes" fields:"DK"`
	StaffDescription    string `json:"staffDescription" firestore:"staffDescription" fields:"DK"` // instructor classes, midway org
	OABanquet           bool   `json:"oaBanquet" firestore:"oaBanquet" fields:"DK"`

	InstructorClasses []int  `json:"instructorClasses" firestore:"instructorClasses"`
	Notes             string `json:"notes" firestore:"notes" fields:""`
	NoShow            bool   `json:"noShow" firestore:"noShow" feilds:""`

	LastUpdateTime time.Time `json:"lastUpdateTime" firestore:"lastUpdateTime,serverTimestamp"`
}

// Type returns a short description of the participant's registration type.
func (p *Participant) Type() string {
	switch {
	case p.Staff:
		return "Staff"
	case p.Youth:
		return "Youth"
	default:
		return "Adult"
	}
}

func (p *Participant) Unit() string {
	if p.UnitNumber == "" {
		return p.UnitType
	}
	return p.UnitType + " " + p.UnitNumber
}

func (p *Participant) Name() string {
	if p.Suffix != "" {
		return p.FirstName + " " + p.LastName + " " + p.Suffix
	}
	return p.FirstName + " " + p.LastName
}

func (p *Participant) NicknameOrFirstName() string {
	if p.Nickname != "" {
		return p.Nickname
	}
	return p.FirstName
}

// Firsts returns Name's or Nickname's.
func (p *Participant) Firsts() string {
	n := p.NicknameOrFirstName()
	if strings.HasSuffix(n, "s") {
		return n + "'"
	}
	return n + "'s"
}

func (p *Participant) LookupLunch(c *Conference) *Lunch {
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
	// XXX check instructor classes
	for _, class := range p.Classes {
		if l, ok := c.lunch.byClass[class]; ok {
			return l
		}
	}
	if l, ok := c.lunch.byUnitType[p.UnitType]; ok {
		return l
	}
	if len(c.Lunches) == 0 {
		return &Lunch{Seating: 1, Location: "TBD"}
	}
	return c.Lunches[0]
}
