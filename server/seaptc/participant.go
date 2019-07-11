package main

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/seaptc/server/model"
	"github.com/seaptc/server/store"
	"golang.org/x/sync/errgroup"

	"github.com/garyburd/web/httperror"
	"github.com/garyburd/web/templates"
)

type participantService struct {
	*application
	templates struct {
		Error  *templates.Template `html:"participant/error.html participant/root.html common.html"`
		Closed *templates.Template `html:"participant/closed.html participant/root.html common.html"`
		Home   *templates.Template `html:"participant/home.html blurbs.html participant/root.html common.html"`
		Login  *templates.Template `html:"participant/login.html participant/root.html common.html"`
		Eval1  *templates.Template `html:"participant/eval1.html participant/root.html common.html"`
		Eval2  *templates.Template `html:"participant/eval2.html participant/root.html common.html"`
	}
}

func (svc *participantService) init(ctx context.Context, a *application, tm *templates.Manager) error {
	svc.application = a
	tm.NewFromFields(&svc.templates)
	return nil
}

func (svc *participantService) errorTemplate() *templates.Template {
	return svc.templates.Error
}

func (svc *participantService) makeHandler(v interface{}) func(*requestContext) error {
	f, ok := v.(func(*participantService, *requestContext) error)
	if !ok {
		return nil
	}
	return func(rc *requestContext) error {
		return f(svc, rc)
	}
}

const (
	stateBefore = iota
	stateOpen
	stateGrace
	stateAfter
)

func (svc *participantService) serviceState(rc *requestContext) (int, *model.Conference, error) {
	conf, err := svc.store.GetCachedConference(rc.context())
	if err != nil {
		return 0, nil, err
	}

	const (
		openDuration  = 24 * time.Hour
		graceDuration = 48 * time.Hour
	)

	now := time.Now()

	// Time override for testing.
	if rc.isStaff {
		var s string
		svc.debugTimeCodec.Decode(rc.request, &s)
		switch s {
		case "before":
			now = svc.conferenceDate.Add(-time.Hour)
		case "open":
			now = svc.conferenceDate.Add(time.Hour)
		case "grace":
			now = svc.conferenceDate.Add(openDuration + time.Hour)
		case "after":
			now = svc.conferenceDate.Add(graceDuration + time.Hour)
		}
	}

	since := now.Sub(svc.conferenceDate)
	state := stateAfter
	switch {
	case since < 0:
		state = stateBefore
	case since < 24*time.Hour:
		state = stateOpen
	case since < 48:
		state = stateGrace
	}
	return state, conf, nil
}

func (svc *participantService) Serve_(rc *requestContext) error {
	if rc.request.URL.Path != "/" {
		rc.participantName = ""
		return httperror.ErrNotFound
	}

	state, conf, err := svc.serviceState(rc)
	if err != nil {
		return err
	}
	if state == stateBefore || state == stateAfter ||
		(state == stateGrace && rc.participantID == "") {
		rc.participantName = ""
		return svc.serveHomeClosed(rc, conf, state)
	}

	if rc.participantID == "" || rc.request.Method == "POST" || rc.request.FormValue("loginCode") != "" {
		rc.participantName = ""
		return svc.serveHomeLogin(rc, conf)
	}

	return svc.serveHome(rc, conf)
}

func (svc *participantService) serveHome(rc *requestContext, conf *model.Conference) error {
	var data struct {
		Participant         *model.Participant
		Conference          *model.Conference
		SessionClasses      []*model.SessionClass
		Lunch               *model.Lunch
		EvaluatedClasses    []*model.SessionClass
		EvaluatedConference bool
	}

	classMaps, err := svc.store.GetCachedClassMaps(rc.context())
	if err != nil {
		return err
	}

	data.Conference, err = svc.store.GetCachedConference(rc.context())
	if err != nil {
		return err
	}

	var g errgroup.Group

	g.Go(func() error {
		var err error
		data.Participant, err = svc.store.GetParticipant(rc.context(), rc.participantID)
		if err != nil {
			return err
		}
		data.SessionClasses = classMaps.ParticipantSessionClasses(data.Participant)

		data.Lunch = conf.ParticipantLunch(data.Participant)
		return nil
	})

	g.Go(func() error {
		var err error
		data.EvaluatedConference, data.EvaluatedClasses, err = svc.store.GetRecordedEvaluations(
			rc.context(), rc.participantID, classMaps)
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	return rc.respond(svc.templates.Home, http.StatusOK, &data)
}

func (svc *participantService) serveHomeLogin(rc *requestContext, conf *model.Conference) error {
	loginCode := rc.request.FormValue("loginCode")
	participant, err := svc.store.GetParticipantForLoginCode(rc.context(), loginCode)
	switch {
	case err == nil:
		svc.participantIDCodec.Encode(rc.response, participant.ID, participant.Name())
		http.Redirect(rc.response, rc.request, "/", http.StatusSeeOther)
		return nil
	case err != store.ErrNotFound:
		return err
	}

	var data = struct {
		Participant *model.Participant
		Conference  *model.Conference
		LoginCode   string
		Invalid     bool
	}{
		nil,
		conf,
		loginCode,
		rc.request.Method == "POST" || loginCode != "",
	}
	return rc.respond(svc.templates.Login, http.StatusOK, &data)
}

func (svc *participantService) serveHomeClosed(rc *requestContext, conf *model.Conference, state int) error {
	var data = struct {
		Before bool
	}{
		state == stateBefore,
	}
	return rc.respond(svc.templates.Closed, http.StatusOK, &data)
}

func (svc *participantService) Serve_logout(rc *requestContext) error {
	svc.participantIDCodec.Encode(rc.response, nil)
	http.Redirect(rc.response, rc.request, "/", http.StatusSeeOther)
	return nil
}

func (svc *participantService) Serve_eval(rc *requestContext) error {
	state, _, err := svc.serviceState(rc)
	if err != nil {
		return err
	}
	if state == stateBefore || state == stateAfter || rc.participantID == "" {
		http.Redirect(rc.response, rc.request, "/", http.StatusSeeOther)
		return nil
	}

	rc.request.ParseForm()
	data := struct {
		Form               url.Values
		Invalid            map[string]string
		SessionClass       *model.SessionClass
		EvaluateClass      bool
		EvaluateConference bool
	}{
		Form:    rc.request.Form,
		Invalid: make(map[string]string),
	}

	evaluationCode := strings.TrimSpace(rc.request.FormValue("evaluationCode"))

	if evaluationCode == "conference" {
		data.EvaluateConference = true
	} else {
		classMaps, err := svc.store.GetCachedClassMaps(rc.context())
		if err != nil {
			return err
		}
		data.SessionClass = classMaps.SessionClassByEvaluationCode[evaluationCode]
		if data.SessionClass == nil {
			if rc.request.FormValue("submit") != "" {
				data.Invalid["evaluationCode"] = "Class not found"
			}
			return rc.respond(svc.templates.Eval1, http.StatusOK, &data)
		}
		data.EvaluateClass = true
		data.EvaluateConference = data.SessionClass.Session == model.NumSession-1
	}

	if rc.request.Method != "POST" {
		// Respond with form filled with previously entered feedback.

		if data.EvaluateClass {
			classEvaluation, err := svc.store.GetClassEvaluation(rc.context(), rc.participantID, data.SessionClass.Session)
			switch {
			case err == store.ErrNotFound:
				// do nothing
			case err != nil:
				return err
			case classEvaluation.Class == data.SessionClass.Number:
				setClassEvaluationForm(data.Form, classEvaluation, "")
			}
		}

		if data.EvaluateConference {
			conferenceEvaluation, err := svc.store.GetConferenceEvaluation(rc.context(), rc.participantID)
			switch {
			case err == store.ErrNotFound:
				// do nothing
			case err != nil:
				return err
			default:
				setConferenceEvaluationForm(data.Form, conferenceEvaluation)
			}
		}

		return rc.respond(svc.templates.Eval2, http.StatusOK, &data)
	}

	getRating := func(name string, required bool) int {
		n, _ := strconv.Atoi(rc.request.FormValue(name))
		if required && (n < 1 || n > 4) {
			data.Invalid[name] = "is-invalid"
		}
		return n
	}

	var classEvaluation *model.ClassEvaluation
	if data.EvaluateClass {
		classEvaluation = &model.ClassEvaluation{
			ParticipantID:      rc.participantID,
			Session:            data.SessionClass.Session,
			Class:              data.SessionClass.Number,
			Updated:            time.Now().In(model.TimeLocation),
			Source:             "online",
			KnowledgeRating:    getRating("knowledge", true),
			PresentationRating: getRating("presentation", true),
			UsefulnessRating:   getRating("usefulness", true),
			OverallRating:      getRating("overall", true),
			Comments:           strings.TrimSpace(rc.request.FormValue("comments")),
		}
	}

	var conferenceEvaluation *model.ConferenceEvaluation
	if data.EvaluateConference {
		conferenceEvaluation = &model.ConferenceEvaluation{
			ParticipantID:           rc.participantID,
			Updated:                 time.Now().In(model.TimeLocation),
			Source:                  "online",
			ExperienceRating:        getRating("experience", false),
			PromotionRating:         getRating("promotion", false),
			RegistrationRating:      getRating("registration", false),
			CheckinRating:           getRating("checkin", false),
			MidwayRating:            getRating("midway", false),
			LunchRating:             getRating("lunch", false),
			FacilitiesRating:        getRating("facilities", false),
			SignageWayfindingRating: getRating("signageWayfinding", false),
			WebsiteRating:           getRating("website", false),
			LearnTopics:             strings.TrimSpace(rc.request.FormValue("learnTopics")),
			TeachTopics:             strings.TrimSpace(rc.request.FormValue("teachTopics")),
			Comments:                strings.TrimSpace(rc.request.FormValue("overallComments")),
		}
	}

	if len(data.Invalid) > 0 {
		return rc.respond(svc.templates.Eval2, http.StatusOK, &data)
	}

	if classEvaluation != nil {
		err := svc.store.SetClassEvaluation(rc.context(), classEvaluation)
		if err != nil {
			return err
		}
	}

	if conferenceEvaluation != nil {
		err := svc.store.SetConferenceEvaluation(rc.context(), conferenceEvaluation)
		if err != nil {
			return err
		}
	}

	switch {
	case classEvaluation != nil && conferenceEvaluation != nil:
		return rc.redirect("/", "info", "Evaluation recorded for session %d and the conference.", classEvaluation.Session+1)
	case classEvaluation != nil:
		return rc.redirect("/", "info", "Evaluation recorded for session %d.", classEvaluation.Session+1)
	case conferenceEvaluation != nil:
		return rc.redirect("/", "info", "Evaluation recorded for the conference.")
	default:
		http.Redirect(rc.response, rc.request, "/", http.StatusSeeOther)
		return nil
	}
}

func ratingString(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func setClassEvaluationForm(form url.Values, e *model.ClassEvaluation, suffix string) {
	form.Set("knowledge"+suffix, ratingString(e.KnowledgeRating))
	form.Set("presentation"+suffix, ratingString(e.PresentationRating))
	form.Set("usefulness"+suffix, ratingString(e.UsefulnessRating))
	form.Set("overall"+suffix, ratingString(e.OverallRating))
	form.Set("comments"+suffix, e.Comments)
}

func setConferenceEvaluationForm(form url.Values, e *model.ConferenceEvaluation) {
	form.Set("experience", ratingString(e.ExperienceRating))
	form.Set("promotion", ratingString(e.PromotionRating))
	form.Set("onlineRegistration", ratingString(e.RegistrationRating))
	form.Set("checkin", ratingString(e.CheckinRating))
	form.Set("midway", ratingString(e.MidwayRating))
	form.Set("lunch", ratingString(e.LunchRating))
	form.Set("facilities", ratingString(e.FacilitiesRating))
	form.Set("website", ratingString(e.WebsiteRating))
	form.Set("signageWayfinding", ratingString(e.SignageWayfindingRating))
	form.Set("learnTopics", e.LearnTopics)
	form.Set("teachTopics", e.TeachTopics)
	form.Set("overallComments", e.Comments)
}
