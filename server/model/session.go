package model

import (
	"log"
)

type ClassMap map[int]*Class

func NewClassMap(classes []*Class) ClassMap {
	m := make(map[int]*Class)
	for _, c := range classes {
		m[c.Number] = c
	}
	return m
}

type SessionClass struct {
	*Class
	Session    int
	Part       int
	Instructor bool
}

var noClass = &Class{Title: "No Class", Length: 1}

func (classes ClassMap) ParticipantSessionClasses(p *Participant) []*SessionClass {
	sessionClasses := make([]*SessionClass, NumSession)
	for i := range sessionClasses {
		sessionClasses[i] = &SessionClass{Session: i, Class: noClass}
	}
	for _, n := range p.Classes {
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
			sc.Class = c
			sc.Part = i - start + 1
		}
	}
	for _, ic := range p.InstructorClasses {
		c := classes[ic.Class]
		if c == nil {
			log.Printf("unknown instructor class %d for participant %v", ic.Class, p.ID)
			continue
		}
		if ic.Session < 0 || ic.Session >= NumSession {
			log.Printf("bad instructor session %d for participant %v", ic.Session, p.ID)
			continue
		}
		sc := sessionClasses[ic.Session]
		sc.Class = c
		sc.Part = c.Start() - ic.Session + 1
		sc.Instructor = true
	}
	return sessionClasses
}
