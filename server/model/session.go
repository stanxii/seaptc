package model

import "log"

type SessionConflict struct {
	*Class
	Instructor bool
}

type SessionClass struct {
	*Class
	Session    int
	Instructor bool
	Conflicts  []SessionConflict
}

var noClass = &Class{Title: "No Class", Length: 1}

func ParticipantSessionClasses(p *Participant, classes map[int]*Class) []*SessionClass {
	sessionClasses := make([]*SessionClass, NumSession)
	for i := range sessionClasses {
		sessionClasses[i] = &SessionClass{Session: i, Class: noClass}
	}
	setSessionClasses := func(classNumbers []int, instructor bool) {
		for _, n := range classNumbers {
			c := classes[n]
			if c == nil {
				log.Printf("unknown class %d for participant %v", n, p.ID)
				continue
			}
			start, end := c.StartEnd()
			if end >= len(sessionClasses) {
				continue
			}
			for i := start; i <= end; i++ {
				sc := sessionClasses[i]
				if sc.Class.Number != 0 {
					sc.Conflicts = append(sc.Conflicts, SessionConflict{Class: sc.Class, Instructor: sc.Instructor})
				}
				sc.Class = c
				sc.Instructor = instructor
			}
		}
	}
	setSessionClasses(p.Classes, false)
	// XXX Handle instructor classes
	return sessionClasses
}
