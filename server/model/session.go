package model

import (
	"fmt"
	"log"
	"strings"
)

type SessionClass struct {
	*Class
	Session    int
	Part       int
	Instructor bool
}

type ClassMaps struct {
	ClassByNumber                map[int]*Class
	SessionClassByEvaluationCode map[string]*SessionClass
}

func NewClassMaps(classes []*Class) *ClassMaps {
	cms := &ClassMaps{
		ClassByNumber:                make(map[int]*Class),
		SessionClassByEvaluationCode: make(map[string]*SessionClass),
	}
	for _, c := range classes {
		cms.ClassByNumber[c.Number] = c
		for i, code := range strings.Split(c.EvaluationCodes, ",") {
			code = strings.TrimSpace(code)
			cms.SessionClassByEvaluationCode[code] = &SessionClass{
				Class:   c,
				Session: c.Start() + i,
				Part:    i + 1,
			}
		}
	}
	return cms
}

func (sc *SessionClass) NumberDotPart() string {
	if sc.Length <= 1 {
		return fmt.Sprintf("%d", sc.Number)
	}
	return fmt.Sprintf("%d.%d", sc.Number, sc.Part)
}

func (sc *SessionClass) IofN() string {
	if sc.Length <= 1 {
		return ""
	}
	return fmt.Sprintf(" (%d of %d)", sc.Part, sc.Length)
}

func (sc *SessionClass) EvaluationCode() string {
	codes := strings.Split(sc.EvaluationCodes, ",")
	if 1 <= sc.Part && sc.Part <= len(codes) {
		return strings.TrimSpace(codes[sc.Part-1])
	}
	return ""
}

var noClass = &Class{Title: "No Class", Length: 1}

func (cms *ClassMaps) ParticipantSessionClasses(p *Participant) []*SessionClass {
	sessionClasses := make([]*SessionClass, NumSession)
	for i := range sessionClasses {
		sessionClasses[i] = &SessionClass{Session: i, Class: noClass}
	}
	for _, n := range p.Classes {
		c := cms.ClassByNumber[n]
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
		c := cms.ClassByNumber[ic.Class]
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
		sc.Part = ic.Session - c.Start() + 1
		sc.Instructor = true
	}
	return sessionClasses
}
