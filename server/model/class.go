package model

import (
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/datastore"
)

//go:generate go run gogen.go -input class.go -output gen_class.go Class

// Class represents a PTC class.
//
// The Import tag marks fields that are copied from the planning spreadsheet.
type Class struct {
	Number int `json:"number" datastore:"_" fields:"Import"`

	Length         int    `json:"length" datastore:"length" fields:"Import"`
	Responsibility string `json:"responsibility"  datastore:"responsibility" fields:"Import"`
	New            string `json:"new" datastore:"new,noindex" fields:"Import"`
	Title          string `json:"title" datastore:"title" fields:"Import"`
	TitleNote      string `json:"titleNote" datastore:"titleNote,noindex" fields:"Import"`
	Description    string `json:"description" datastore:"description,noindex" fields:"Import"`
	Programs       int    `json:"programs" datastore:"programs,noindex" fields:"Import"`
	Capacity       int    `json:"capacity" datastore:"capacity" fields:"Import"`
	Location       string `json:"location" datastore:"location" fields:"Import"`
	SpreadsheetRow int    `json:"-" datastore:"spreadsheetRow,noindex" fields:"Import"`
	AccessToken    string `json:"accessToken" datastore:"accessToken,noindex" fields:"Import"`

	// The following fields are comma separated strings.
	InstructorNames  string `json:"instructorNames" datastore:"instructorNames" fields:"Import"`
	InstructorEmails string `json:"instructorEmails" datastore:"instructorEmails" fields:"Import"`
	EvaluationCodes  string `json:"evaluationCodes" datastore:"evaluationCodes" fields:"Import"`

	// Hash computed from planning spreadhseet fields.
	ImportHash string `datastore:"importHash"`
}

// init initializes derived fields.
func (c *Class) init() {
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

func (c *Class) ShortTitle() string {
	if i := strings.Index(c.Title, " - "); i > 0 {
		return c.Title[:i]
	}
	if strings.HasSuffix(c.Title, ")") {
		if i := strings.Index(c.Title, " ("); i > 0 {
			return c.Title[:i]
		}
	}
	return c.Title
}

func SplitComma(s string) []string {
	var result []string

	for len(s) > 0 {
		s = strings.TrimLeft(s, " ")
		i := strings.IndexByte(s, ',')
		t := s
		if i < 0 {
			i = len(s)
		}
		t = s[:i]
		s = s[i+1:]
		t = strings.TrimRight(t, " ")
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}

func IsValidClassNumber(number int) bool {
	return 100 <= number && number < (NumSession+1)*100
}

func SortClasses(classes []*Class, key string) {
	key, reverse := SortKeyReverse(key)
	switch key {
	case Class_Location:
		sort.Slice(classes, reverse(func(i, j int) bool {
			switch {
			case classes[i].Location < classes[j].Location:
				return true
			case classes[i].Location > classes[j].Location:
				return false
			default:
				return classes[i].Number < classes[j].Number
			}
		}))
	case Class_Responsibility:
		sort.Slice(classes, reverse(func(i, j int) bool {
			switch {
			case classes[i].Responsibility < classes[j].Responsibility:
				return true
			case classes[i].Responsibility > classes[j].Responsibility:
				return false
			default:
				return classes[i].Number < classes[j].Number
			}
		}))
	case Class_Capacity:
		sort.Slice(classes, reverse(func(i, j int) bool {
			switch {
			case classes[i].Capacity < classes[j].Capacity:
				return true
			case classes[i].Capacity > classes[j].Capacity:
				return false
			default:
				return classes[i].Number < classes[j].Number
			}
		}))
	default:
		sort.Slice(classes, reverse(func(i, j int) bool { return classes[i].Number < classes[j].Number }))
	}
}

var deletedClassFields = map[string]bool{}

func (c *Class) Load(ps []datastore.Property) error {
	err := datastore.LoadStruct(c, filterProperties(ps, deletedClassFields))
	if err != nil {
		return err
	}
	c.init()
	return nil
}

func (c *Class) LoadKey(k *datastore.Key) error {
	c.Number = int(k.ID)
	return nil
}

func (c *Class) Save() ([]datastore.Property, error) {
	ps, err := datastore.SaveStruct(c)
	return ps, err
}
