package model

type SSClass struct {
	Number   int  `json:"number"`
	Elective bool `json:"elective"`
}

type SuggestedSchedule struct {
	Program int       `json:"program"`
	Name    string    `json:"name"`
	Classes []SSClass `json:"classes"`
}
