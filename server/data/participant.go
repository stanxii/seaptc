package data

import "strings"

type Participant struct {
	RegistrationNumber  string `json:"registrationNumber" firestore:"registrationNumber" merge:"DK"`
	FirstName           string `json:"firstName" firestore:"firstName" merge:"DK"`
	LastName            string `json:"lastName" firestore:"lastName" merge:"DK"`
	Nickname            string `json:"nickname" firestore:"nickname" merge:"DK"`
	Suffix              string `json:"suffix" firestore:"suffix" merge:"DK"`
	Staff               bool   `json:"staff" firestore:"staff" merge:"DK"`
	Youth               bool   `json:"youth" firestore:"youth" merge:"DK"`
	Phone               string `json:"phone" firestore:"phone" merge:"DK"`
	Email               string `json:"email" firestore:"email" merge:"DK"`
	Address             string `json:"address" firestore:"address" merge:"DK"`
	City                string `json:"city" firestore:"city" merge:"DK"`
	State               string `json:"state" firestore:"state" merge:"DK"`
	Zip                 string `json:"zip" firestore:"zip" merge:"DK"`
	StaffRole           string `json:"staffRole" firestore:"staffRole" merge:"DK"` // Instructor, Support, Midway
	Council             string `json:"council" firestore:"council" merge:"DK"`
	District            string `json:"district" firestore:"district" merge:"DK"`
	UnitType            string `json:"unitType" firestore:"unitType" merge:"DK"`
	UnitNumber          string `json:"unitNumber" firestore:"unitNumber" merge:"DK"`
	DietaryRestrictions string `json:"dietaryRestrictions" firestore:"dietaryRestrictions" merge:"DK"`
	Marketing           string `json:"marketing" firestore:"marketing" merge:"DK"`
	ScoutingYears       string `json:"scoutingYears" firestore:"scoutingYears" merge:"DK"`
	ShowQRCode          bool   `json:"showQRCode" firestore:"showQRCode" merge:"DK"`
	BSANumber           string `json:"bsaNumber" firestore:"bsaNumber" merge:"DK"`
	Classes             []int  `json:"classes" firestore:"classes" merge:"DK"`
	StaffDescription    string `json:"staffDescription" firestore:"staffDescription" merge:"DK"` // instructor classes, midway org
	OABanquet           bool   `json:"oaBanquet" firestore:"oaBanquet" merge:"DK"`

	InstructorClasses []int  `json:"instructorClasses" firestore:"instructorClasses"`
	Notes             string `json:"notes" firestore:"notes" merge:""`
	NoShow            bool   `json:"noShow" firestore:"noShow" merge:""`
}

// DocName returns the document name for Firestore.
func (p *Participant) DocName() string {
	ya := "a"
	if p.Youth {
		ya = "y"
	}
	return strings.Join([]string{p.LastName, p.FirstName, p.Suffix, ya, p.RegistrationNumber}, "_")
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
