package data

import (
	"log"
	"time"
)

var TimeLocation = timeLocation()

func timeLocation() *time.Location {
	l, err := time.LoadLocation("US/Pacific")
	if err != nil {
		log.Fatal(err)
	}
	return l
}
