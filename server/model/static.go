package model

import (
	"log"
	"strings"
	"time"
)

const (
	NumSession   = 6
	LunchSession = 2

	// Special classses.
	NoClassClassNumber   = 999
	OABanquetClassNumber = 700
)

type Session struct {
	Start time.Duration
	End   time.Duration
	Lunch bool
}

var Sessions = []*Session{
	{
		9 * time.Hour,
		10 * time.Hour,
		false,
	},
	{
		10*time.Hour + 10*time.Minute,
		11*time.Hour + 10*time.Minute,
		false,
	},
	{
		11*time.Hour + 20*time.Minute,
		13*time.Hour + 15*time.Minute,
		true,
	},
	{
		13*time.Hour + 25*time.Minute,
		14*time.Hour + 25*time.Minute,
		false,
	},
	{
		14*time.Hour + 35*time.Minute,
		15*time.Hour + 35*time.Minute,
		false,
	},
	{
		15*time.Hour + 45*time.Minute,
		16*time.Hour + 45*time.Minute,
		false,
	},
}

var TimeLocation = timeLocation()

func timeLocation() *time.Location {
	l, err := time.LoadLocation("US/Pacific")
	if err != nil {
		log.Fatal(err)
	}
	return l
}

const (
	CubScoutProgram = iota
	ScoutsBSAProgram
	VenturingProgram
	SeaScoutProgram
	CommissionerProgram
	YouthProgram
	NumPrograms
)

type ProgramDescription struct {
	Code string
	Name string
}

func (pd *ProgramDescription) TitleName() string {
	return strings.Title(pd.Name)
}

var ProgramDescriptions = []*ProgramDescription{
	// Must match order in xxxPorgram constants above.
	{"cub", "Cub Pack adults"},
	{"bsa", "Scout Troop adults"},
	{"ven", "Venturing Crew adults"},
	{"sea", "Sea Scout adults"},
	{"com", "Commissioners"},
	{"you", "youth"},

	// AllProgram must be last in slice for programDescriptionsForMask()
	{"all", "everyone"},
}

func programDescriptionsForMask(mask int, reverse bool) []*ProgramDescription {
	if (1<<NumPrograms)-1 == mask {
		return ProgramDescriptions[NumPrograms:]
	}

	var result []*ProgramDescription
	for i := 0; i < NumPrograms; i++ {
		if ((1 << uint(i)) & mask) != 0 {
			result = append(result, ProgramDescriptions[i])
		}
	}

	if reverse {
		i := 0
		j := len(result) - 1
		for i < j {
			result[i], result[j] = result[j], result[i]
			i++
			j--
		}
	}

	return result
}
