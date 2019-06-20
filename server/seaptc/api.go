package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"

	"github.com/seaptc/server/dk"
	"github.com/seaptc/server/model"
	"github.com/seaptc/server/store"
)

type apiService struct {
	*application
	templates struct {
		Error *templates.Template `text:"api/error.json"`
	}
}

func (svc *apiService) init(ctx context.Context, a *application, tm *templates.Manager) error {
	svc.application = a
	tm.NewFromFields(&svc.templates)
	return nil
}

func (svc *apiService) errorTemplate() *templates.Template {
	return svc.templates.Error
}

func (svc *apiService) makeHandler(v interface{}) func(*requestContext) error {
	f, ok := v.(func(*apiService, *requestContext) error)
	if !ok {
		return nil
	}
	return func(rc *requestContext) error { return f(svc, rc) }
}

func (svc *apiService) Serve_api_(rc *requestContext) error {
	return httperror.ErrNotFound
}

func (svc *apiService) respond(rc *requestContext, data interface{}) error {
	data = map[string]interface{}{"result": data}
	p, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		return err
	}
	rc.response.Header().Set("Content-Type", "application/json")
	rc.response.Header().Set("Content-Length", strconv.Itoa(len(p)))
	if rc.request.Method != "HEAD" {
		rc.response.Write(p)
	}
	return nil
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

var wsPattern = regexp.MustCompile(`[\r\n\t ]+`)

func createSpecialSessionEvent(number int, title string, start, end time.Duration, conf *model.Conference) *sessionEvent {
	description := ""
	if i := strings.Index(title, "\n"); i >= 0 {
		description = title[i+1:]
		title = title[:i]
	}
	return &sessionEvent{
		Number:      number,
		Title:       strings.TrimSpace(wsPattern.ReplaceAllLiteralString(title, " ")),
		Description: strings.TrimSpace(wsPattern.ReplaceAllLiteralString(description, " ")),
		StartTime:   []int{conf.Year, conf.Month, conf.Day, int(start / time.Hour), int((start % time.Hour) / time.Minute)},
		EndTime:     []int{conf.Year, conf.Month, conf.Day, int(end / time.Hour), int((end % time.Hour) / time.Minute)},
	}
}

func (svc *apiService) Serve_api_sessionEvents_(rc *requestContext) error {
	numString := strings.TrimPrefix(rc.request.URL.Path, "/api/sessionEvents/")
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

	return svc.respond(rc, se)
}

func (svc *apiService) Serve_api_uploadRegistrationsToken(rc *requestContext) error {
	return svc.respond(rc, map[string]interface{}{
		"name":  "_xsrftoken",
		"value": rc.xsrfToken("/api/uploadRegistrations"),
	})
}

func (svc *apiService) Serve_api_uploadRegistrations(rc *requestContext) error {
	if rc.request.Method != "POST" {
		return httperror.ErrMethodNotAllowed
	}
	if !rc.isAdmin {
		return httperror.ErrForbidden
	}

	f, _, err := rc.request.FormFile("file")
	if err == http.ErrMissingFile {
		return &httperror.Error{Status: 400, Message: "Export file not uploaded"}
	}
	if err != nil {
		return err
	}
	defer f.Close()

	participants, err := dk.ParseCSV(f)
	if err != nil {
		return err
	}

	n, err := svc.store.ImportParticipants(rc.context(), participants)
	if err != nil {
		return err
	}

	return svc.respond(rc, map[string]interface{}{"count": n})
}
