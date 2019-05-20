package model

import (
	"fmt"
	"sort"
	"time"
)

//go:generate go run gogen.go -input class.go -output gen_class.go Class

// Class represents a PTC class.
//
// The dk tag marks fields that contribute to the Doubleknot session event
// definition. The nosheet tag marks fields that should not be copied from the
// planning spreadsheet.
type Class struct {
	Number           int      `json:"number" firestore:"number" fields:"Sheet"`
	Length           int      `json:"length" firestore:"length" fields:"Sheet"`
	Responsibility   string   `json:"responsibility"  firestore:"responsibility" fields:"Sheet"`
	New              string   `json:"new" firestore:"new" fields:"Sheet"`
	Title            string   `json:"title" firestore:"title" fields:"Sheet"`
	TitleNotes       string   `json:"titleNotes" firestore:"titleNotes" fields:"Sheet"`
	Description      string   `json:"description" firestore:"description" fields:"Sheet"`
	Programs         int      `json:"programs" firestore:"programs" fields:"Sheet"`
	Capacity         int      `json:"capacity" firestore:"capacity" fields:"Sheet"`
	Location         string   `json:"location" firestore:"location" fields:"Sheet"`
	SpreadsheetRow   int      `json:"-" firestore:"spreadsheetRow" fields:"Sheet"`
	InstructorNames  []string `json:"instructorNames" firestore:"instructorNames" fields:"Sheet"`
	InstructorEmails []string `json:"instructorEmails" firestore:"instructorEmails" fields:"Sheet"`
	EvaluationCodes  []string `json:"evaluationCodes" firestore:"evaluationCodes" fields:"Sheet"`
	AccessToken      string   `json:"accessToken" firestore:"accessToken" fields:"Sheet"`

	// DKHash is the hash of the fields last set on the Doubleknot session event description.
	DKHash bool `json:"dkHash" firestore:"dkHash"`

	LastUpdateTime time.Time `json:"lastUpdateTime" firestore:"lastUpdateTime,serverTimestamp"`
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

func (c *Class) ProgramDescriptions() []*ProgramDescription {
	return programDescriptionsForMask(c.Programs)
}

func SortedClasses(m map[int]*Class, what string) []*Class {
	classes := make([]*Class, len(m))
	i := 0
	for _, class := range m {
		classes[i] = class
		i++
	}

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
