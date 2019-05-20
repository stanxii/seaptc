package sheet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/seaptc/server/model"
	"golang.org/x/oauth2/google"
	"golang.org/x/xerrors"
)

type class struct {
	model.Class
}

var setters = []struct {
	name string
	fn   func(*class, string) error
}{
	{"XXXX Num", func(c *class, s string) error { return setInt(&c.Number, s) }},
	{"Class Length (# Sns)", func(c *class, s string) error { return setInt(&c.Length, s) }},
	{"Responsiblity", func(c *class, s string) error { return setString(&c.Responsibility, s) }},
	{"New!", func(c *class, s string) error { return setString(&c.New, s) }},
	{"Title", func(c *class, s string) error { return setString(&c.Title, s) }},
	{"Title notes", func(c *class, s string) error { return setString(&c.TitleNotes, s) }},
	{"Description", func(c *class, s string) error { return setString(&c.Description, s) }},
	{"NSCC Location", func(c *class, s string) error { return setString(&c.Location, s) }},
	{"XXXX Instructor Confirmed", setInstructors},
	{"XXXX Email", func(c *class, s string) error { return setList(&c.InstructorEmails, strings.ToLower(s)) }},
	{"Evaluation Codes", func(c *class, s string) error { return setList(&c.EvaluationCodes, s) }},
	{"Dashboard Code", func(c *class, s string) error { return setString(&c.AccessToken, s) }},
	{"Cub", func(c *class, s string) error { return setProgram(c, 1<<model.CubScoutProgram, s) }},
	{"BS", func(c *class, s string) error { return setProgram(c, 1<<model.ScoutsBSAProgram, s) }},
	{"Ven", func(c *class, s string) error { return setProgram(c, 1<<model.VenturingProgram, s) }},
	{"Sea Scouts", func(c *class, s string) error { return setProgram(c, 1<<model.SeaScoutProgram, s) }},
	{"Com", func(c *class, s string) error { return setProgram(c, 1<<model.CommissionerProgram, s) }},
	{"Youth", func(c *class, s string) error { return setProgram(c, 1<<model.YouthProgram, s) }},
	{"ALL", func(c *class, s string) error { return setProgram(c, (1<<model.NumPrograms)-1, s) }},
	{"Requested Capacity", setCapacity},
	{"Actual Location & Registration Capacity", setCapacity},
}

var (
	listDelimPattern       = regexp.MustCompile(`[\t\r\n;, ]+`)
	wsPattern              = regexp.MustCompile(`[\r\n\t ]+`)
	parenPattern           = regexp.MustCompile(`\([^(]*\)`)
	instructorDelimPattern = regexp.MustCompile(`[\r\n\t ]*[/,][\r\n\t ]*`)
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

func setList(pv *[]string, s string) error {
	var v []string
	for _, e := range listDelimPattern.Split(s, -1) {
		if e != "" {
			v = append(v, e)
		}
	}
	*pv = v
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
	c.InstructorNames = v
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

func parseSheet(r io.Reader) ([]*model.Class, error) {
	var sheet struct {
		Rows [][]string `json:"values"`
	}
	if err := json.NewDecoder(r).Decode(&sheet); err != nil {
		return nil, err
	}

	if len(sheet.Rows) < 2 {
		return nil, errors.New("could not find header row")
	}

	header := sheet.Rows[1]
	if len(header) < 1 || len(header[0]) < 4 {
		return nil, fmt.Errorf("could not find class number column header")
	}

	// Create map of column header name to column index. Replace year number
	// with XXXX.
	year := header[0][:4]
	columnIndex := map[string]int{}
	for j, name := range header {
		if strings.HasPrefix(name, year) {
			name = "XXXX" + name[len(year):]
		}
		columnIndex[name] = j
	}
	for _, s := range setters {
		if _, ok := columnIndex[s.name]; !ok {
			return nil, fmt.Errorf("could not find column %s in sheet", s.name)
		}
	}

	var result []*model.Class
	for i := 2; i < len(sheet.Rows); i++ {
		row := sheet.Rows[i]
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
		if c.Number == 700 {
			// Ignore OA banquet
			continue
		}
		start, end := c.StartEnd()
		if start >= model.NumSession || end >= model.NumSession {
			return nil, fmt.Errorf("class %d has bad number or length (%d)", c.Number, c.Length)
		}
		result = append(result, &c.Class)
	}
	return result, nil
}

func Fetch(ctx context.Context, config *model.AppConfig) ([]*model.Class, error) {
	jwtConfig, err := google.JWTConfigFromJSON([]byte(config.PlanningSheetServiceAccountKey), "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		return nil, xerrors.Errorf("error parsing planning sheet service account key: %w", err)
	}
	client := jwtConfig.Client(ctx)
	resp, err := client.Get(config.PlanningSheetURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("fetch sheet returned %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	return parseSheet(resp.Body)
}
