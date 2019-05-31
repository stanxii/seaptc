package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"

	"github.com/seaptc/server/model"
	"github.com/seaptc/server/store"
)

type sessionEventsService struct {
	*application
	templates struct {
		Error *templates.Template `text:"sessionevents/error.json"`
	}
}

func (svc *sessionEventsService) init(ctx context.Context, a *application, tm *templates.Manager) error {
	svc.application = a
	tm.NewFromFields(&svc.templates)
	return nil
}

func (svc *sessionEventsService) errorTemplate() *templates.Template {
	return svc.templates.Error
}

func (svc *sessionEventsService) makeHandler(v interface{}) func(*requestContext) error {
	f, ok := v.(func(*sessionEventsService, *requestContext) error)
	if !ok {
		return nil
	}
	return func(rc *requestContext) error { return f(svc, rc) }
}

type sessionEvent struct {
	Number       int      `json:"number"`
	Title        string   `json:"title"`
	New          string   `json:"titleNew"` // rename to avoid js reserved word
	TitleNote    string   `json:"titleNote"`
	Description  string   `json:"description"`
	StartSession int      `json:"startSession"`
	EndSession   int      `json:"endSession"`
	StartTime    []int    `json:"startTime"` // year, month, day, hour, minute
	EndTime      []int    `json:"endTime"`   // year, month, day, hour, minute
	Capacity     int      `json:"capacity"`  // 0: no limit, -1 no space
	Programs     []string `json:"programs"`
}

func createSessionEvent(conf *model.Conference, class *model.Class) *sessionEvent {
	start := model.Sessions[class.Start()].Start
	startHour := start / time.Hour
	startMinute := (start - time.Hour*startHour) / time.Minute

	end := model.Sessions[class.End()].End
	endHour := end / time.Hour
	endMinute := (end - time.Hour*endHour) / time.Minute

	var programs []string
	if class.Programs != (1<<model.NumPrograms)-1 {
		for _, pd := range class.ProgramDescriptions(false) {
			programs = append(programs, pd.Name)
		}
	}

	return &sessionEvent{
		Number:       class.Number,
		New:          class.New,
		Title:        class.Title,
		TitleNote:    class.TitleNote,
		Description:  class.Description,
		StartSession: class.Start() + 1,
		EndSession:   class.End() + 1,
		StartTime:    []int{conf.Year, conf.Month, conf.Day, int(startHour), int(startMinute)},
		EndTime:      []int{conf.Year, conf.Month, conf.Day, int(endHour), int(endMinute)},
		Capacity:     class.Capacity,
		Programs:     programs,
	}
}

func (svc *sessionEventsService) handleCORS(rc *requestContext) bool {
	h := rc.response.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "*")
	if rc.request.Method != "OPTIONS" {
		return false
	}
	rc.response.WriteHeader(http.StatusNoContent)
	return true
}

func (svc *sessionEventsService) respond(rc *requestContext, status int, result interface{}) error {
	rc.response.Header().Set("Content-Type", "application/json")
	if rc.request.Method == "HEAD" {
		return nil
	}
	data := map[string]interface{}{"result": result}
	p, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		return err
	}
	rc.response.Write(p)
	return nil
}

func (svc *sessionEventsService) Serve_session__events(rc *requestContext) error {
	if svc.handleCORS(rc) {
		return nil
	}

	classes, err := svc.store.GetAllClassesFull(rc.context())
	if err != nil {
		return err
	}
	conf, err := svc.store.GetConference(rc.context())
	if err != nil {
		return err
	}

	var result []*sessionEvent
	for _, class := range classes {
		result = append(result, createSessionEvent(conf, class))
	}
	return svc.respond(rc, http.StatusOK, result)
}

func (svc *sessionEventsService) Serve_session__events_(rc *requestContext) error {
	if svc.handleCORS(rc) {
		return nil
	}

	numString := strings.TrimPrefix(rc.request.URL.Path, "/session-events/")

	number, err := strconv.Atoi(numString)
	if err != nil || number <= 0 {
		return &httperror.Error{Status: http.StatusNotFound, Message: fmt.Sprintf("Class %q not found.", numString)}
	}

	class, err := svc.store.GetClass(rc.context(), number)
	if err == store.ErrNotFound {
		return &httperror.Error{
			Status: http.StatusNotFound, Message: fmt.Sprintf("Class %q not found.", numString),
			Err: err,
		}
	} else if err != nil {
		return err
	}

	conf, err := svc.store.GetConference(rc.context())
	if err != nil {
		return err
	}

	return svc.respond(rc, http.StatusOK, createSessionEvent(conf, class))
}
