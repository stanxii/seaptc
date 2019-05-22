package model

import (
	"fmt"
	"sort"
)

//go:generate go run gogen.go -input class.go -output gen_class.go Class

// Class represents a PTC class.
//
// The DK tag marks fields that contribute to the Doubleknot session event
// definition. The Sheet tag marks fields that are copied from the planning
// spreadsheet.
type Class struct {
	Number           int      `json:"number" datastore:"number" fields:"Sheet"`
	Length           int      `json:"length" datastore:"length" fields:"Sheet,DK"`
	Responsibility   string   `json:"responsibility"  datastore:"responsibility" fields:"Sheet"`
	New              string   `json:"new" datastore:"new,noindex" fields:"Sheet,DK"`
	Title            string   `json:"title" datastore:"title" fields:"Sheet,DK"`
	TitleNotes       string   `json:"titleNotes" datastore:"titleNotes,noindex" fields:"Sheet,DK"`
	Description      string   `json:"description" datastore:"description,noindex" fields:"Sheet,DK"`
	Programs         int      `json:"programs" datastore:"programs,noindex" fields:"Sheet,DK"`
	Capacity         int      `json:"capacity" datastore:"capacity" fields:"Sheet,DK"`
	Location         string   `json:"location" datastore:"location" fields:"Sheet"`
	SpreadsheetRow   int      `json:"-" datastore:"spreadsheetRow,noindex" fields:"Sheet"`
	InstructorNames  []string `json:"instructorNames" datastore:"instructorNames,noindex" fields:"Sheet"`
	InstructorEmails []string `json:"instructorEmails" datastore:"instructorEmails,noindex" fields:"Sheet"`
	EvaluationCodes  []string `json:"evaluationCodes" datastore:"evaluationCodes,noindex" fields:"Sheet"`
	AccessToken      string   `json:"accessToken" datastore:"accessToken,noindex" fields:"Sheet"`

	DKNeedsUpdate bool `json:"-" datastore:"dkNeedsUpdate"`
}

// Start returns zero based index of the starting session.
func (c *Class) Start() int { return int(c.Number/100) - 1 }

// End returns zero based index of the ending session.
func (c *Class) End() int { return c.Start() + c.Length - 1 }

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

func (c *Class) ProgramDescriptions(reverse bool) []*ProgramDescription {
	return programDescriptionsForMask(c.Programs, reverse)
}

func SortClasses(classes []*Class, what string) []*Class {
	switch what {
	case Class_Location:
		sort.Slice(classes, func(i, j int) bool { return classes[i].Location < classes[j].Location })
	case Class_Responsibility:
		sort.Slice(classes, func(i, j int) bool { return classes[i].Responsibility < classes[j].Responsibility })
	case Class_Capacity:
		sort.Slice(classes, func(i, j int) bool { return classes[i].Capacity < classes[j].Capacity })
	default:
		sort.Slice(classes, func(i, j int) bool { return classes[i].Number < classes[j].Number })
	}
	return classes
}
