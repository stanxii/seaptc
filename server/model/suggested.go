package model

type SSClass struct {
	Number   int
	Elective bool
}

type SuggestedSchedule struct {
	Program int
	Name    string
	Classes []SSClass
}
