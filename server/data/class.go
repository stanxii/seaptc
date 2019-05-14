package data

import (
	"fmt"
	"strconv"
)

const (
	CubScoutProgram = 1 << iota
	ScoutsBSAProgram
	VenturingProgram
	SeaScoutProgram
	CommissionerProgram
	YouthProgram
	AllProgram = CubScoutProgram | ScoutsBSAProgram | VenturingProgram | SeaScoutProgram | CommissionerProgram | YouthProgram
)

type ProgramInfo struct {
	mask int
	Name string
	Slug string
}

var ProgramInfos = []*ProgramInfo{
	{CubScoutProgram, "Cub Pack adults", "cub"},
	{ScoutsBSAProgram, "Scout Troop adults", "bsa"},
	{VenturingProgram, "Venture Crew adults", "ven"},
	{SeaScoutProgram, "Sea Scout adults", "sea"},
	{CommissionerProgram, "Commissioners", "com"},
	{YouthProgram, "Youth", "you"},
	{AllProgram, "Everyone", "all"}, // All must be last in slice for Class.ReverseProgramInfos.
}

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

	// DKDirty is set to true when the class needs updating on Doubleknot.
	DKDirty bool `json:"-" firestore:"dkDirty"`
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

func (c *Class) ReverseProgramInfos() []*ProgramInfo {
	// AllProgram is at end of slice.
	if c.Programs == AllProgram {
		return ProgramInfos[len(ProgramInfos)-1:]
	}

	var result []*ProgramInfo

	// Don't include AllProgram located at end of ProgramInfos slice.
	for i := len(ProgramInfos) - 2; i >= 0; i-- {
		if c.Programs&ProgramInfos[i].mask != 0 {
			result = append(result, ProgramInfos[i])
		}
	}
	return result
}
