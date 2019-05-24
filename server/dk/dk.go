package dk

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/seaptc/server/model"
)

const oaClassNumber = 700

var (
	unitNumberPat      = regexp.MustCompile(`(\d+)`)
	classNumberPattern = regexp.MustCompile(`^(\d\d\d):`)
	commaSepPat        = regexp.MustCompile(`\s*,\s*`)
)

type participant struct {
	model.Participant
	registeredByFirstName string
	registeredByLastName  string
	registrationType      string
	midwayDescription     string
	instructorDescription string
}

var setters = []struct {
	name string
	fn   func(p *participant, s string)
}{
	{"Registration Number", func(p *participant, s string) { p.RegistrationNumber = s }},
	{"Registered By First Name", func(p *participant, s string) { p.registeredByFirstName = s }},
	{"Registered By Last Name", func(p *participant, s string) { p.registeredByLastName = s }},
	{"First Name", func(p *participant, s string) { p.FirstName = s }},
	{"Last Name", func(p *participant, s string) { p.LastName = s }},
	//XXX {"Nickname", func(p *participant, s string) { p.Nickname = s }},
	{"Suffix", func(p *participant, s string) { p.Suffix = s }},
	{"Generic 1", func(p *participant, s string) { p.BSANumber = s }},
	{"Type", func(p *participant, s string) { p.registrationType = s }},
	{"Telephone", func(p *participant, s string) { p.Phone = s }},
	{"Email", func(p *participant, s string) { p.Email = s }},
	{"Address", func(p *participant, s string) { p.Address = s }},
	{"City", func(p *participant, s string) { p.City = s }},
	{"State", func(p *participant, s string) { p.State = s }},
	{"Postal Code", func(p *participant, s string) { p.Zip = s }},
	{"Council", func(p *participant, s string) { p.Council = s }},
	{"District", func(p *participant, s string) { p.District = s }},
	{"Unit Type", func(p *participant, s string) { p.UnitType = s }},
	{"Unit Number", func(p *participant, s string) { p.UnitNumber = s }},
	{"Staff role", func(p *participant, s string) { p.StaffRole = s }},
	{"How many years have you been in scouting?", func(p *participant, s string) { p.ScoutingYears = s }},
	{"Print QR code on PTC name badge?", func(p *participant, s string) { p.ShowQRCode = s == "Yes" }},

	// addDietaryRestriction assumes that Vegan is parsed before Vegetarian.
	{"Do you have any meal requirements?:Vegan", addDietaryRestriction}, // addDietaryRestriction requires Vegan before Vegetarian
	{"Do you have any meal requirements?:Vegetarian", addDietaryRestriction},
	{"Do you have any meal requirements?:Gluten Free", addDietaryRestriction},

	// Downstream code assumes that Other is last marketing option.
	{"How did you hear about the PTC?:Roundtable/District", addMarketing},
	{"How did you hear about the PTC?:eTotem", addMarketing},
	{"How did you hear about the PTC?:Council website", addMarketing},
	{"How did you hear about the PTC?:Attended before", addMarketing},
	{"How did you hear about the PTC?:Wood Badge", addMarketing},
	{"What other ways did you hear about the PTC?", addMarketing},

	{"Which classes are you teaching?", func(r *participant, s string) { r.instructorDescription = s }},
	{"Which organization are you representing on the midway?", func(r *participant, s string) { r.midwayDescription = s }},
}

func addDietaryRestriction(p *participant, s string) {
	if s == "" {
		return
	}
	if s == "Vegetarian" && strings.Contains(p.DietaryRestrictions, "Vegan") {
		return
	}
	if p.DietaryRestrictions == "" {
		p.DietaryRestrictions = s
	}
	p.DietaryRestrictions = p.DietaryRestrictions + "; " + s
}

func addMarketing(p *participant, s string) {
	if s == "" {
		return
	}
	if p.Marketing == "" {
		p.Marketing = s
		return
	}
	p.Marketing = p.Marketing + "; " + strings.Replace(s, ";", " ", -1)
}

func ParseCSV(rd io.Reader) ([]*model.Participant, error) {

	// Skip BOM

	var bom [3]byte
	if _, err := io.ReadFull(rd, bom[:]); err != nil {
		return nil, err
	}

	csvr := csv.NewReader(rd)

	header, err := csvr.Read()
	if err != nil {
		return nil, fmt.Errorf("dk: error reading header: %v", err)
	}

	columnIndex := map[string]int{}
	for j, name := range header {
		columnIndex[name] = j
	}
	for _, s := range setters {
		if _, ok := columnIndex[s.name]; !ok {
			return nil, fmt.Errorf("could not find column %q in export file", s.name)
		}
	}
	eventColumnIndex, ok := columnIndex["Event Name"]
	if !ok {
		errors.New("could not find Event Name column in export file")
	}

	// Process body rows.

	var (
		participants []*model.Participant
		p            *participant
	)
	for i := 1; ; i++ {
		row, err := csvr.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		event := row[eventColumnIndex]
		if m := classNumberPattern.FindStringSubmatch(event); m != nil {
			if p == nil {
				return nil, errors.New("dk: found class row before PTC row")
			}
			n, _ := strconv.Atoi(m[1])
			if n == oaClassNumber {
				p.OABanquet = true
			} else {
				p.Classes = append(p.Classes, n)
			}
		} else if !strings.HasSuffix(event, "Program and Training Conference") {
			return nil, errors.New("dk: event not XXX: or PTC")
		} else {
			p = &participant{}
			participants = append(participants, &p.Participant)
			for _, s := range setters {
				j := columnIndex[s.name]
				if j >= len(row) {
					return nil, errors.New("dk: short row")
				}
				cell := strings.TrimSpace(row[j])
				s.fn(p, cell)
			}
			cleanParticipant(p)
		}
	}
	for _, p := range participants {
		sort.Ints(p.Classes)
	}
	return participants, nil
}

func titleCase(s string) string {
	// Use s if s is mixed case.
	ls := strings.ToLower(s)
	if s != ls && s != strings.ToUpper(s) {
		return s
	}
	return strings.Title(ls)
}

func titleCase2(s string, r string) string {
	// Use s if s is mixed case.
	ls := strings.ToLower(s)
	if s != ls && s != strings.ToUpper(s) {
		return s
	}

	// User r if lower(r) == lower(s) and r is mixed case.
	lr := strings.ToLower(r)
	if lr == ls && r != lr && r != strings.ToUpper(r) {
		return r
	}

	return strings.Title(ls)
}

var removeSuffix = map[string]bool{
	"MBA":  true,
	"Esq.": true,
}

func cleanParticipant(p *participant) {
	p.FirstName = titleCase2(p.FirstName, p.registeredByFirstName)
	p.LastName = titleCase2(p.LastName, p.registeredByLastName)
	p.Nickname = titleCase(p.Nickname)

	if p.Nickname == p.FirstName {
		// Remove trivial nickname
		p.Nickname = ""
	} else if i := len(p.Nickname) - len(p.LastName); i > 0 && p.Nickname[i:] == p.LastName && p.Nickname[i-1] == ' ' {
		// Remove last name.
		p.Nickname = p.Nickname[:i-1]

		if i := len(p.Nickname) - len(p.FirstName); i >= 0 && p.Nickname[:len(p.FirstName)] == p.FirstName {
			if i == 0 {
				// Remainder is first name, no need for nickname.
				p.Nickname = ""
			} else if i == 2 && p.Nickname[len(p.Nickname)-2] == ' ' {
				// Remainder is first name, middle initial. No need for nickname.
				p.Nickname = ""
			}
		}
	}

	if removeSuffix[p.Suffix] {
		p.Suffix = ""
	}

	p.City = titleCase(p.City)
	p.Email = strings.ToLower(p.Email)
	p.UnitNumber = strings.TrimLeft(unitNumberPat.FindString(p.UnitNumber), "0")

	if i := strings.Index(p.District, " ("); i > 0 {
		p.District = p.District[:i]
	} else if p.District != "Council" {
		p.District = ""
	}

	p.Youth = strings.Contains(p.registrationType, "Youth")
	p.Staff = strings.Contains(p.registrationType, "Staff")
	if p.Staff {
		if i := strings.Index(p.StaffRole, " ("); i > 0 {
			p.StaffRole = p.StaffRole[:i]
		}
	} else {
		p.StaffRole = ""
	}

	if p.Council == "Other" {
		p.Council = ""
	}

	if p.Council != "Chief Seattle" {
		p.District = ""
	}

	if p.UnitType == "Council" || p.UnitType == "District" {
		p.UnitNumber = ""
	}

	switch p.StaffRole {
	case "Midway":
		p.StaffDescription = p.midwayDescription
	case "Instructor":
		p.StaffDescription = p.instructorDescription
	}

	// Shorten "Cub Pack" to "Pack", etc.
	if i := strings.LastIndex(p.UnitType, " "); i >= 0 {
		p.UnitType = p.UnitType[i+1:]
	}
}
