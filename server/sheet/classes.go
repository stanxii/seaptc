package sheet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/seaptc/server/model"
)

const oaClassNumber = "700"

type class struct {
	model.Class
}

var setters = []struct {
	name string
	fn   func(*class, string) error
}{
	{"number", func(c *class, s string) error { return setInt(&c.Number, s) }},
	{"length", func(c *class, s string) error { return setInt(&c.Length, s) }},
	{"responsibility", func(c *class, s string) error { return setString(&c.Responsibility, s) }},
	{"new", func(c *class, s string) error { return setString(&c.New, s) }},
	{"title", func(c *class, s string) error { return setString(&c.Title, s) }},
	{"titleNote", func(c *class, s string) error { return setString(&c.TitleNote, s) }},
	{"description", func(c *class, s string) error { return setString(&c.Description, s) }},
	{"location", func(c *class, s string) error { return setString(&c.Location, s) }},
	{"instructorNames", setInstructors},
	{"instructorEmails", func(c *class, s string) error { return setList(&c.InstructorEmails, strings.ToLower(s)) }},
	{"evaluationCodes", func(c *class, s string) error { return setString(&c.EvaluationCodes, s) }},
	{"accessToken", func(c *class, s string) error { return setString(&c.AccessToken, s) }},
	{"cub", func(c *class, s string) error { return setProgram(c, 1<<model.CubScoutProgram, s) }},
	{"bsa", func(c *class, s string) error { return setProgram(c, 1<<model.ScoutsBSAProgram, s) }},
	{"ven", func(c *class, s string) error { return setProgram(c, 1<<model.VenturingProgram, s) }},
	{"sea", func(c *class, s string) error { return setProgram(c, 1<<model.SeaScoutProgram, s) }},
	{"com", func(c *class, s string) error { return setProgram(c, 1<<model.CommissionerProgram, s) }},
	{"you", func(c *class, s string) error { return setProgram(c, 1<<model.YouthProgram, s) }},
	{"all", func(c *class, s string) error { return setProgram(c, (1<<model.NumPrograms)-1, s) }},
	{"requestedCapacity", setCapacity},
	{"locationCapacity", setCapacity},
}

var (
	listDelimPattern       = regexp.MustCompile(`[\t\r\n;, ]+`)
	wsPattern              = regexp.MustCompile(`[\r\n\t ]+`)
	parenPattern           = regexp.MustCompile(`\([^(]*\)`)
	instructorDelimPattern = regexp.MustCompile(`[\r\n\t ]*[/,][\r\n\t ]*`)
	classNumberPattern     = regexp.MustCompile(`^\s*\d\d\d\s*$`)
)

func setString(pv *string, s string) error {
	*pv = s
	return nil
}

func setInt(pv *int, s string) error {
	var v int
	if s != "" {
		var err error
		v, err = strconv.Atoi(s)
		if err != nil {
			return err
		}
	}
	*pv = v
	return nil
}

func setList(pv *string, s string) error {
	var v []string
	for _, e := range listDelimPattern.Split(s, -1) {
		if e != "" {
			v = append(v, e)
		}
	}
	sort.Strings(v)
	*pv = strings.Join(v, ", ")
	return nil
}

func setInstructors(c *class, s string) error {
	var v []string
	s = parenPattern.ReplaceAllLiteralString(s, " ")
	for _, e := range instructorDelimPattern.Split(s, -1) {
		if e != "" {
			v = append(v, strings.TrimSpace(e))
		}
	}
	sort.Strings(v)
	c.InstructorNames = strings.Join(v, ", ")
	return nil
}

func setProgram(c *class, mask int, s string) error {
	if s == "" {
		return nil
	}
	c.Programs |= mask
	return nil
}

func setCapacity(c *class, s string) error {
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	if v == 0 {
		return nil
	}
	if c.Capacity == 0 || v < c.Capacity {
		c.Capacity = v
	}
	return nil
}

func parseClasses(r io.Reader) ([]*model.Class, error) {
	var sheet struct {
		Rows [][]string `json:"values"`
	}
	if err := json.NewDecoder(r).Decode(&sheet); err != nil {
		return nil, err
	}

	if len(sheet.Rows) < 1 {
		return nil, errors.New("could not find header row")
	}

	header := sheet.Rows[0]
	columnIndex := map[string]int{}
	for j, name := range header {
		name = strings.TrimSpace(name)
		if name != "" {
			columnIndex[name] = j
		}
	}
	for _, s := range setters {
		if _, ok := columnIndex[s.name]; !ok {
			return nil, fmt.Errorf("could not find column %s in sheet", s.name)
		}
	}

	var result []*model.Class
	for i := 1; i < len(sheet.Rows); i++ {
		row := sheet.Rows[i]
		if len(row) < 1 || !classNumberPattern.MatchString(row[0]) || row[0] == oaClassNumber {
			continue
		}
		var c class
		for _, s := range setters {
			j := columnIndex[s.name]
			if j >= len(row) {
				continue
			}
			cell := strings.TrimSpace(wsPattern.ReplaceAllLiteralString(row[j], " "))
			if err := s.fn(&c, cell); err != nil {
				return nil, fmt.Errorf("sheet (%d, %s): %v", i, s.name, err)
			}
		}
		start, end := c.StartEnd()
		if start >= model.NumSession || end >= model.NumSession {
			return nil, fmt.Errorf("class %d has bad number or length (%d)", c.Number, c.Length)
		}
		result = append(result, &c.Class)
	}
	return result, nil
}
