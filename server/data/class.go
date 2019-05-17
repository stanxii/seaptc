package data

import (
	"fmt"
	"strconv"
)

// Class represents a PTC class.
//
// The dk tag marks fields that contribute to the Doubleknot session event
// definition. The nosheet tag marks fields that should not be copied from the
// planning spreadsheet.
type Class struct {
	Number           int      `json:"number" firestore:"number" merge:"Sheet"`
	Length           int      `json:"length" firestore:"length" merge:"Sheet"`
	Responsibility   string   `json:"responsibility"  firestore:"responsibility" merge:"Sheet"`
	New              string   `json:"new" firestore:"new" merge:"Sheet"`
	Title            string   `json:"title" firestore:"title" merge:"Sheet"`
	TitleNotes       string   `json:"titleNotes" firestore:"titleNotes" merge:"Sheet"`
	Description      string   `json:"description" firestore:"description" merge:"Sheet"`
	Programs         int      `json:"programs" firestore:"programs" merge:"Sheet"`
	Capacity         int      `json:"capacity" firestore:"capacity" merge:"Sheet"`
	Location         string   `json:"location" firestore:"location" merge:"Sheet"`
	SpreadsheetRow   int      `json:"-" firestore:"spreadsheetRow" merge:"Sheet"`
	InstructorNames  []string `json:"instructorNames" firestore:"instructorNames" merge:"Sheet"`
	InstructorEmails []string `json:"instructorEmails" firestore:"instructorEmails" merge:"Sheet"`
	EvaluationCodes  []string `json:"evaluationCodes" firestore:"evaluationCodes" merge:"Sheet"`
	AccessToken      string   `json:"accessToken" firestore:"accessToken" merge:"Sheet"`

	// DKHash is the hash of the fields last set on the Doubleknot session event description.
	DKHash bool `json:"dkHash" firestore:"dkHash"`
}

// DocName returns the document name for Firestore
func (c *Class) DocName() string { return strconv.Itoa(c.Number) }

// Start returns zero based index of the starting session.
func (c *Class) Start() int { return int(c.Number/100) - 1 }

// StartEnd returns zero based indexes of first session and last session of
// class.
func (c *Class) StartEnd() (int, int) {
	start := c.Start()
	end := start + c.Length - 1
	return start, end
}

// FormatPart returns the empty string for classes of length one and
// fmt.Sprintf(format, part) for classes with length greater than one.
func (c *Class) FormatPart(format string, session int) string {
	if c.Number == 0 || c.Length <= 1 {
		return ""
	}
	return fmt.Sprintf(format, session-c.Start()+1)
}

// FormatPartLength returns the empty string for classes of length one and
// fmt.Sprintf(format, part, length) for classes with length greater than one.
func (c *Class) PartOfLength(format string, session int) string {
	if c.Number == 0 || c.Length <= 1 {
		return ""
	}
	return fmt.Sprintf(format, session-c.Start()+1, c.Length)
}

const (
	CubScoutProgram = 1 << iota
	ScoutsBSAProgram
	VenturingProgram
	SeaScoutProgram
	CommissionerProgram
	YouthProgram
	AllProgram = CubScoutProgram | ScoutsBSAProgram | VenturingProgram | SeaScoutProgram | CommissionerProgram | YouthProgram
)

type ProgramDescription struct {
	mask int
	Slug string
	Name string
}

var ProgramDescriptions = []*ProgramDescription{
	{CubScoutProgram, "cub", "Cub Pack adults"},
	{ScoutsBSAProgram, "bsa", "Scout Troop adults"},
	{VenturingProgram, "ven", "Venture Crew adults"},
	{SeaScoutProgram, "sea", "Sea Scout adults"},
	{CommissionerProgram, "com", "Commissioners"},
	{YouthProgram, "you", "Youth"},

	// AllProgram must be last in slice for ReverseProgramInfos()
	{AllProgram, "all", "Everyone"},
}

func (c *Class) ProgramDescriptions() []*ProgramDescription {
	// AllProgram is at end of slice.
	if c.Programs == AllProgram {
		return ProgramDescriptions[len(ProgramDescriptions)-1:]
	}

	var result []*ProgramDescription

	// Don't include AllProgram located at end of ProgramInfos slice.
	// Return in reverse order for convenient layout of images in HTML.
	for i := len(ProgramDescriptions) - 2; i >= 0; i-- {
		if c.Programs&ProgramDescriptions[i].mask != 0 {
			result = append(result, ProgramDescriptions[i])
		}
	}
	return result
}

type Classes struct {
	Slice []*Class       // Sorted by class number
	Map   map[int]*Class // Key is class number
}
