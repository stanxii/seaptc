package sheet

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/seaptc/server/model"
)

var programs = map[string]int{
	"Cub Scouts":   model.CubScoutProgram,
	"Scouts BSA":   model.ScoutsBSAProgram,
	"Venturing":    model.VenturingProgram,
	"Sea Scouts":   model.SeaScoutProgram,
	"Commissioner": model.CommissionerProgram,
	"Youth":        model.YouthProgram,
}

func parseSuggestedSchedules(r io.Reader) ([]*model.SuggestedSchedule, error) {
	var result []*model.SuggestedSchedule

	var sheet struct {
		Rows [][]string `json:"values"`
	}
	if err := json.NewDecoder(r).Decode(&sheet); err != nil {
		return nil, err
	}

	for i, row := range sheet.Rows {
		if len(row) < 3 {
			continue
		}
		program, ok := programs[row[0]]
		if !ok {
			continue
		}
		var ss model.SuggestedSchedule
		ss.Program = program
		ss.Name = row[1]
		for j := 2; j < len(row); j += 2 {
			s := strings.TrimSpace(row[j])
			if s == "" {
				continue
			}
			n, err := strconv.Atoi(s)
			if err != nil {
				return nil, fmt.Errorf("sheet: could not parse class number %q in row %d, column %d", row[j], i+1, j+1)
			}
			elective := false
			if j+1 < len(row) {
				elective, _ = strconv.ParseBool(strings.TrimSpace(row[j+1]))
			}
			ss.Classes = append(ss.Classes, model.SSClass{Number: n, Elective: elective})
		}
		result = append(result, &ss)
	}
	return result, nil
}
