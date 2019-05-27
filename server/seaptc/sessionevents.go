package main

import (
	"context"
	"encoding/json"
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
		Error *templates.Template `text:"sessionevents/error.txt"`
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
	Number      int      `json:"number"`
	New         string   `json:"titleNew"` // avoid js reserved word
	Title       string   `json:"title"`
	TitleNote   string   `json:"titleNote"`
	Description string   `json:"description"`
	StartTime   []int    `json:"startTime"` // year, monthIndex, day, hour, minute
	EndTime     []int    `json:"endTime"`   // year, monthIndex, day, hour, minute
	Capacity    int      `json:"capacity"`  // 0: no limit, -1 no space
	Programs    []string `json:"programs"`
}

func createSessionEvent(conf *model.Conference, class *model.Class) *sessionEvent {
	start := model.Sessions[class.Start()].Start
	startHour := start / time.Hour
	startMinute := (start - time.Hour*startHour) / time.Minute

	end := model.Sessions[class.End()].End
	endHour := end / time.Hour
	endMinute := (end - time.Hour*endHour) / time.Minute

	var programs []string
	for _, pd := range class.ProgramDescriptions(false) {
		programs = append(programs, pd.Name)
	}

	return &sessionEvent{
		Number:      class.Number,
		New:         class.New,
		Title:       class.Title,
		TitleNote:   class.TitleNote,
		Description: class.Description,
		StartTime:   []int{conf.Year, conf.Month - 1, conf.Day, int(startHour), int(startMinute)},
		EndTime:     []int{conf.Year, conf.Month - 1, conf.Day, int(endHour), int(endMinute)},
		Programs:    programs,
	}
}

func (svc *sessionEventsService) handleSessionEventsCORS(rc *requestContext) bool {
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

func (svc *sessionEventsService) Serve_sessionEventsService_sessionevents(rc *requestContext) error {
	if svc.handleSessionEventsCORS(rc) {
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

	result := make([]*sessionEvent, len(classes))
	for i, class := range classes {
		result[i] = createSessionEvent(conf, class)
	}

	rc.response.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(rc.response).Encode(result)
}

func (svc *sessionEventsService) Serve_session__events_(rc *requestContext) error {
	if svc.handleSessionEventsCORS(rc) {
		return nil
	}

	number, err := strconv.Atoi(strings.TrimPrefix(rc.request.URL.Path, "/session-events"))
	if err != nil {
		return httperror.ErrNotFound
	}

	class, err := svc.store.GetClass(rc.context(), number)
	if err == store.ErrNotFound {
		err = httperror.ErrNotFound
	}

	conf, err := svc.store.GetConference(rc.context())
	if err != nil {
		return err
	}

	rc.response.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(rc.response).Encode(createSessionEvent(conf, class))
}
