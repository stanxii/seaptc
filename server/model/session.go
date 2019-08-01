package model

import (
	"fmt"
	"log"
	"sync"
)

type SessionClass struct {
	*Class
	Session    int
	Instructor bool
}

func (sc *SessionClass) NumberDotPart() string {
	if sc.Length <= 1 {
		return fmt.Sprintf("%d", sc.Number)
	}
	return fmt.Sprintf("%d.%d", sc.Number, sc.part())
}

func (sc *SessionClass) IofN() string {
	if sc.Length <= 1 {
		return ""
	}
	return fmt.Sprintf(" (%d of %d)", sc.part(), sc.Length)
}

func (sc *SessionClass) part() int {
	return sc.Session - sc.Start() + 1
}

func (sc *SessionClass) EvaluationCode() string {
	codes := SplitComma(sc.EvaluationCodes)
	i := sc.Session - sc.Start()
	if 0 <= i && i < len(codes) {
		return codes[i]
	}
	return ""
}

type ClassInfo struct {
	classes []*Class
	number  map[int]*Class

	evalCode struct {
		once  sync.Once
		value map[string]*SessionClass
	}

	sessions struct {
		once  sync.Once
		value [][]*SessionClass
	}
}

func NewClassInfo(classes []*Class) *ClassInfo {
	SortClasses(classes, "")
	ci := &ClassInfo{
		classes: classes,
		number:  make(map[int]*Class),
	}
	for _, c := range classes {
		ci.number[c.Number] = c
	}
	return ci
}

func (ci *ClassInfo) Classes() []*Class {
	return ci.classes
}

func (ci *ClassInfo) LookupNumber(number int) *Class {
	return ci.number[number]
}

func (ci *ClassInfo) LookupEvaluationCode(evaluationCode string) *SessionClass {
	ci.evalCode.once.Do(func() {
		ci.evalCode.value = make(map[string]*SessionClass)
		for _, c := range ci.classes {
			for i, code := range SplitComma(c.EvaluationCodes) {
				ci.evalCode.value[code] = &SessionClass{Class: c, Session: c.Start() + i}
			}
		}
	})
	return ci.evalCode.value[evaluationCode]
}

func (ci *ClassInfo) Sessions() [][]*SessionClass {
	ci.sessions.once.Do(func() {
		ci.sessions.value = make([][]*SessionClass, NumSession)
		for _, c := range ci.classes {
			start, end := c.StartEnd()
			for i := start; i <= end; i++ {
				if i >= NumSession {
					continue
				}
				ci.sessions.value[i] = append(ci.sessions.value[i], &SessionClass{Class: c, Session: i})
			}
		}
	})
	return ci.sessions.value
}

var noClass = &Class{Title: "No Class", Length: 1}

func (ci *ClassInfo) ParticipantSessionClasses(p *Participant) []*SessionClass {
	sessionClasses := make([]*SessionClass, NumSession)
	for i := range sessionClasses {
		sessionClasses[i] = &SessionClass{Session: i, Class: noClass}
	}
	for _, n := range p.Classes {
		c := ci.LookupNumber(n)
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
		}
	}
	for _, ic := range p.InstructorClasses {
		c := ci.LookupNumber(ic.Class)
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
		sc.Instructor = true
	}
	return sessionClasses
}
