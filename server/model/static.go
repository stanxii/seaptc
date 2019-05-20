package model

import (
	"log"
	"time"
)

/*
type Session struct {
    StartHour int `json:"startHour" firstore:"startHour"`
    StartHour int `json:"startHour" firstore:"startHour"`
    StartMinute int `json:"StartMinute" firstore:"startMinute"`
    EndHour int `json:"endHour" firstore:"endHour"`
    EndMinute int `json:"endMinute" firstore:"endMinute"`
    Lunch bool `json:"lunch" firstore:"lunch"`
}
*/

var TimeLocation = timeLocation()

func timeLocation() *time.Location {
	l, err := time.LoadLocation("US/Pacific")
	if err != nil {
		log.Fatal(err)
	}
	return l
}

const NumSession = 6

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

var ProgramDescriptions = []*ProgramDescription{
	// Must match order in xxxPorgram constants above.
	{"cub", "Cub Pack adults"},
	{"bsa", "Scout Troop adults"},
	{"ven", "Venture Crew adults"},
	{"sea", "Sea Scout adults"},
	{"com", "Commissioners"},
	{"you", "Youth"},

	// AllProgram must be last in slice for ProgramDescriptions()
	{"all", "Everyone"},
}

func programDescriptionsForMask(mask int) []*ProgramDescription {
	// AllProgram is at end of slice.
	if (1<<NumPrograms)-1 == mask {
		return ProgramDescriptions[NumPrograms:]
	}

	// Don't include AllProgram located at end of ProgramDescriptions slice.
	// Return in reverse order for convenient layout of images in HTML.
	var result []*ProgramDescription
	for i := NumPrograms - 1; i >= 0; i-- {
		if ((1 << uint(i)) & mask) != 0 {
			result = append(result, ProgramDescriptions[i])
		}
	}
	return result
}
