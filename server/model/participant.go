package model

import (
	"strings"
)

//go:generate go run gogen.go -input participant.go -output gen_participant.go Participant

type Participant struct {
	RegistrationNumber  string `json:"registrationNumber" datastore:"registrationNumber,noindex" fields:"DK"`
	FirstName           string `json:"firstName" datastore:"firstName,noindex" fields:"DK"`
	LastName            string `json:"lastName" datastore:"lastName,noindex" fields:"DK"`
	Nickname            string `json:"nickname" datastore:"nickname,noindex" fields:"DK"`
	Suffix              string `json:"suffix" datastore:"suffix,noindex" fields:"DK"`
	Staff               bool   `json:"staff" datastore:"staff,noindex" fields:"DK"`
	Youth               bool   `json:"youth" datastore:"youth,noindex" fields:"DK"`
	Phone               string `json:"phone" datastore:"phone,noindex" fields:"DK"`
	Email               string `json:"email" datastore:"email,noindex" fields:"DK"`
	Address             string `json:"address" datastore:"address,noindex" fields:"DK"`
	City                string `json:"city" datastore:"city,noindex" fields:"DK"`
	State               string `json:"state" datastore:"state,noindex" fields:"DK"`
	Zip                 string `json:"zip" datastore:"zip,noindex" fields:"DK"`
	StaffRole           string `json:"staffRole" datastore:"staffRole,noindex" fields:"DK"` // Instructor, Support, Midway
	Council             string `json:"council" datastore:"council,noindex" fields:"DK"`
	District            string `json:"district" datastore:"district,noindex" fields:"DK"`
	UnitType            string `json:"unitType" datastore:"unitType,noindex" fields:"DK"`
	UnitNumber          string `json:"unitNumber" datastore:"unitNumber,noindex" fields:"DK"`
	DietaryRestrictions string `json:"dietaryRestrictions" datastore:"dietaryRestrictions,noindex" fields:"DK"`
	Marketing           string `json:"marketing" datastore:"marketing,noindex" fields:"DK"`
	ScoutingYears       string `json:"scoutingYears" datastore:"scoutingYears,noindex" fields:"DK"`
	ShowQRCode          bool   `json:"showQRCode" datastore:"showQRCode,noindex" fields:"DK"`
	BSANumber           string `json:"bsaNumber" datastore:"bsaNumber,noindex" fields:"DK"`
	Classes             []int  `json:"classes" datastore:"classes,noindex" fields:"DK"`
	StaffDescription    string `json:"staffDescription" datastore:"staffDescription,noindex" fields:"DK"` // instructor classes, midway org
	OABanquet           bool   `json:"oaBanquet" datastore:"oaBanquet,noindex" fields:"DK"`

	InstructorClasses []int  `json:"instructorClasses" datastore:"instructorClasses,noindex"`
	Notes             string `json:"notes" datastore:"notes,noindex" fields:""`
	NoShow            bool   `json:"noShow" datastore:"noShow,noindex" feilds:""`
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
	c.setupLunch()
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
