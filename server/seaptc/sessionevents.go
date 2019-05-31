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
	end := model.Sessions[class.End()].End

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
		StartTime:    []int{conf.Year, conf.Month, conf.Day, int(start / time.Hour), int((start % time.Hour) / time.Minute)},
		EndTime:      []int{conf.Year, conf.Month, conf.Day, int(end / time.Hour), int((end % time.Hour) / time.Minute)},
		Capacity:     class.Capacity,
		Programs:     programs,
	}
}

func createSpecialSessionEvent(number int, title string, start, end time.Duration, conf *model.Conference) *sessionEvent {
	description := ""
	if i := strings.Index(title, "\n"); i >= 0 {
		description = title[i+1:]
		title = title[:i]
	}
	return &sessionEvent{
		Number:      number,
		Title:       strings.TrimSpace(title),
		Description: strings.TrimSpace(description),
		StartTime:   []int{conf.Year, conf.Month, conf.Day, int(start / time.Hour), int((start % time.Hour) / time.Minute)},
		EndTime:     []int{conf.Year, conf.Month, conf.Day, int(end / time.Hour), int((end % time.Hour) / time.Minute)},
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

func (svc *sessionEventsService) Serve_session__events_(rc *requestContext) error {
	if svc.handleCORS(rc) {
		return nil
	}

	numString := strings.TrimPrefix(rc.request.URL.Path, "/session-events/")
	number, err := strconv.Atoi(numString)
	if err != nil {
		return &httperror.Error{Status: http.StatusNotFound, Message: fmt.Sprintf("Class %q not found.", numString)}
	}

	conf, err := svc.store.GetConference(rc.context())
	if err != nil {
		return err
	}

	var se *sessionEvent
	switch number {
	case model.NoClassClassNumber:
		se = createSpecialSessionEvent(number, conf.NoClassDescription,
			model.Sessions[0].Start, model.Sessions[model.NumSession-1].End,
			conf)
	case model.OABanquetClassNumber:
		se = createSpecialSessionEvent(number, conf.OABanquetDescription,
			17*time.Hour+30*time.Minute,
			21*time.Hour+30*time.Minute,
			conf)
	default:
		class, err := svc.store.GetClass(rc.context(), number)
		if err == store.ErrNotFound {
			return &httperror.Error{
				Status: http.StatusNotFound, Message: fmt.Sprintf("Class %q not found.", numString),
				Err: err,
			}
		} else if err != nil {
			return err
		}
		se = createSessionEvent(conf, class)
	}

	rc.response.Header().Set("Content-Type", "application/json")
	if rc.request.Method == "HEAD" {
		return nil
	}
	data := map[string]interface{}{"result": se}
	p, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		return err
	}
	rc.response.Write(p)
	return nil
}
