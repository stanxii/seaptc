package model

import (
	"fmt"
	"sort"
	"strings"
)

//go:generate go run gogen.go -input participant.go -output gen_participant.go Participant

type Participant struct {
	ID string `json:"id" datastore:"-" fields:""`

	RegistrationNumber  string `json:"registrationNumber" datastore:"regNumber,noindex" fields:"Import"`
	RegisteredByName    string `json:"registeredByName" datastore:"regByName,noindex" fields:"Import"`
	RegisteredByEmail   string `json:"registeredByEmail" datastore:"regByEmail,noindex" fields:"Import"`
	RegisteredByPhone   string `json:"registeredByPhone" datastore:"regByPhone,noindex" fields:"Import"`
	FirstName           string `json:"firstName" datastore:"firstName" fields:"Import,Print"`
	LastName            string `json:"lastName" datastore:"lastName" fields:"Import,Print"`
	Nickname            string `json:"nickname" datastore:"nickname,noindex,omitempty" fields:"Import,Print"`
	Suffix              string `json:"suffix" datastore:"suffix" fields:"Import,Print"`
	Staff               bool   `json:"staff" datastore:"staff" fields:"Import"`
	Youth               bool   `json:"youth" datastore:"youth" fields:"Import"`
	Phone               string `json:"phone" datastore:"phone,noindex,omitempty" fields:"Import"`
	Email               string `json:"email" datastore:"email" fields:"Import"`
	Address             string `json:"address" datastore:"address,noindex,omitempty" fields:"Import"`
	City                string `json:"city" datastore:"city,noindex,omitempty" fields:"Import"`
	State               string `json:"state" datastore:"state,noindex,omitempty" fields:"Import"`
	Zip                 string `json:"zip" datastore:"zip,noindex,omitempty" fields:"Import"`
	StaffRole           string `json:"staffRole" datastore:"staffRole" fields:"Import"` // Instructor, Support, Midway
	Council             string `json:"council" datastore:"council" fields:"Import"`
	District            string `json:"district" datastore:"district" fields:"Import"`
	UnitType            string `json:"unitType" datastore:"unitType" fields:"Import"`
	UnitNumber          string `json:"unitNumber" datastore:"unitNumber" fields:"Import"`
	DietaryRestrictions string `json:"dietaryRestrictions" datastore:"dietaryRestrictions" fields:"Import"`
	Marketing           string `json:"marketing" datastore:"marketing,noindex,omitempty" fields:"Import"`
	ScoutingYears       string `json:"scoutingYears" datastore:"scoutingYears,noindex,omitempty" fields:"Import"`
	ShowQRCode          bool   `json:"showQRCode" datastore:"showQRCode,noindex,omitempty" fields:"Import"`
	BSANumber           string `json:"bsaNumber" datastore:"bsaNumber,noindex,omitempty" fields:"Import"`
	Classes             []int  `json:"classes" datastore:"classes" fields:"Import,Print"`
	StaffDescription    string `json:"staffDescription" datastore:"staffDescription" fields:"Import"` // instructor classes, midway org
	OABanquet           bool   `json:"oaBanquet" datastore:"oaBanquet" fields:"Import,Print"`

	InstructorClasses []int  `json:"instructorClasses" datastore:"instructorClasses,noindex" fields:"Print"`
	Notes             string `json:"notes" datastore:"notes,noindex,omitempty" fields:""`
	NoShow            bool   `json:"noShow" datastore:"noShow,noindex,omitempty" fields:""`

	// Hash computed from Doubleknot registration fields.
	ImportHash string `json:"-" datastore:"importHash"`

	// Set to true when a Print field changes.
	PrintForm bool `json:"-" datastore:"printForm"`

	// Unique seven digit code assigned during import.
	LoginCode string `json:"-" datastore:"loginCode"`

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

func DefaultParticipantLess(a, b *Participant) bool {
	return a.sortName < b.sortName
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

func FilterParticipants(participants []*Participant, fn func(*Participant) bool) []*Participant {
	var result []*Participant
	for _, p := range participants {
		if fn(p) {
			result = append(result, p)
		}
	}
	return result
}
