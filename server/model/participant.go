package model

import (
	"fmt"
	"sort"
	"strings"
)

//go:generate go run gogen.go -input participant.go -output gen_participant.go Participant

type Participant struct {
	ID string `json:"id" datastore:"-" fields:""`

	RegistrationNumber  string `json:"registrationNumber" datastore:"registrationNumber,noindex" fields:"Import"`
	RegisteredByName    string `json:"registeredByName" datastore:"registeredByName,noindex" fields:"Import"`
	RegisteredByEmail   string `json:"registeredByEmail" datastore:"registeredByEmail,noindex" fields:"Import"`
	RegisteredByPhone   string `json:"registeredByPhone" datastore:"registeredByPhone,noindex" fields:"Import"`
	FirstName           string `json:"firstName" datastore:"firstName" fields:"Import"`
	LastName            string `json:"lastName" datastore:"lastName" fields:"Import"`
	Nickname            string `json:"nickname" datastore:"nickname,noindex" fields:"Import"`
	Suffix              string `json:"suffix" datastore:"suffix" fields:"Import"`
	Staff               bool   `json:"staff" datastore:"staff" fields:"Import"`
	Youth               bool   `json:"youth" datastore:"youth" fields:"Import"`
	Phone               string `json:"phone" datastore:"phone,noindex" fields:"Import"`
	Email               string `json:"email" datastore:"email,noindex" fields:"Import"`
	Address             string `json:"address" datastore:"address,noindex" fields:"Import"`
	City                string `json:"city" datastore:"city,noindex" fields:"Import"`
	State               string `json:"state" datastore:"state,noindex" fields:"Import"`
	Zip                 string `json:"zip" datastore:"zip,noindex" fields:"Import"`
	StaffRole           string `json:"staffRole" datastore:"staffRole" fields:"Import"` // Instructor, Support, Midway
	Council             string `json:"council" datastore:"council" fields:"Import"`
	District            string `json:"district" datastore:"district" fields:"Import"`
	UnitType            string `json:"unitType" datastore:"unitType" fields:"Import"`
	UnitNumber          string `json:"unitNumber" datastore:"unitNumber" fields:"Import"`
	DietaryRestrictions string `json:"dietaryRestrictions" datastore:"dietaryRestrictions" fields:"Import"`
	Marketing           string `json:"marketing" datastore:"marketing,noindex" fields:"Import"`
	ScoutingYears       string `json:"scoutingYears" datastore:"scoutingYears,noindex" fields:"Import"`
	ShowQRCode          bool   `json:"showQRCode" datastore:"showQRCode,noindex" fields:"Import"`
	BSANumber           string `json:"bsaNumber" datastore:"bsaNumber,noindex" fields:"Import"`
	Classes             []int  `json:"classes" datastore:"classes" fields:"Import"`
	StaffDescription    string `json:"staffDescription" datastore:"staffDescription" fields:"Import"` // instructor classes, midway org
	OABanquet           bool   `json:"oaBanquet" datastore:"oaBanquet" fields:"Import"`

	InstructorClasses []int  `json:"instructorClasses" datastore:"instructorClasses,noindex"`
	Notes             string `json:"notes" datastore:"notes,noindex" fields:""`
	NoShow            bool   `json:"noShow" datastore:"noShow,noindex" fields:""`

	// Hash computed from Doubleknot registration fields.
	ImportHash string `json:"-" datastore:"importHash"`

	sortName string
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

func (p *Participant) Emails() []string {
	if !p.Youth || p.Email == p.RegisteredByEmail {
		return []string{p.Email}
	}
	return []string{p.RegisteredByEmail, p.Email}
}

// Init initializes derived fields.
func (p *Participant) Init() {
	p.sortName = strings.ToLower(fmt.Sprintf("%s\n%s\n%s", p.LastName, p.FirstName, p.Suffix))
}

func SortParticipants(participants []*Participant, key string) {
	key, reverse := SortKeyReverse(key)
	switch key {
	case "unit", "district", "council":
		sort.Slice(participants, func(i, j int) bool {
			if participants[i].Council != participants[j].Council {
				return participants[i].Council < participants[j].Council
			}
			if participants[i].District != participants[j].District {
				return participants[i].District < participants[j].District
			}
			if participants[i].UnitNumber != participants[j].UnitNumber {
				return participants[i].UnitNumber < participants[j].UnitNumber
			}
			if participants[i].UnitType != participants[j].UnitType {
				return participants[i].UnitType < participants[j].UnitType
			}
			return participants[i].sortName < participants[j].sortName
		})
	default:
		sort.Slice(participants, func(i, j int) bool { return participants[i].sortName < participants[j].sortName })
	}
	reverse(participants)
}
