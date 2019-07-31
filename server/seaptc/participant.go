package main

import (
	"context"
	"fmt"
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
	conf, err := svc.store.GetCachedConference(rc.ctx)
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

	classInfo, err := svc.store.GetCachedClassInfo(rc.ctx)
	if err != nil {
		return err
	}

	data.Conference, err = svc.store.GetCachedConference(rc.ctx)
	if err != nil {
		return err
	}

	var g errgroup.Group

	g.Go(func() error {
		var err error
		data.Participant, err = svc.store.GetParticipant(rc.ctx, rc.participantID)
		if err != nil {
			return err
		}
		data.SessionClasses = classInfo.ParticipantSessionClasses(data.Participant)
		data.Lunch = conf.ParticipantLunch(data.Participant)
		return nil
	})

	g.Go(func() error {
		status, err := svc.store.GetEvaluationStatus(rc.ctx, rc.participantID)
		if err != nil {
			return err
		}
		data.EvaluatedConference = status.Conference
		for i, classNumber := range status.ClassNumbers {
			if classNumber != 0 {
				c := classInfo.LookupNumber(classNumber)
				if c != nil {
					data.EvaluatedClasses = append(data.EvaluatedClasses, &model.SessionClass{
						Class:   c,
						Session: i,
					})
				}
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	return rc.respond(svc.templates.Home, http.StatusOK, &data)
}

func (svc *participantService) serveHomeLogin(rc *requestContext, conf *model.Conference) error {
	loginCode := rc.request.FormValue("loginCode")
	participant, err := svc.store.GetParticipantForLoginCode(rc.ctx, loginCode)
	switch {
	case err == nil:
		svc.participantIDCodec.Encode(rc.response, participant.ID, participant.Name())
		http.Redirect(rc.response, rc.request, "/", http.StatusSeeOther)
		return nil
	case err != store.ErrNotFound:
		return err
	}

	var data = struct {
		Conference *model.Conference
		LoginCode  string
		Invalid    bool
	}{
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

	classInfo, err := svc.store.GetCachedClassInfo(rc.ctx)
	if err != nil {
		return err
	}

	rc.request.ParseForm()
	data := struct {
		Form               url.Values
		Invalid            map[string]string
		SessionClass       *model.SessionClass
		EvaluateSession    bool
		EvaluateConference bool
	}{
		Form:    rc.request.Form,
		Invalid: make(map[string]string),
	}

	evaluationCode := strings.TrimSpace(rc.request.FormValue("evaluationCode"))

	if evaluationCode == "conference" {
		data.EvaluateConference = true
	} else {
		data.SessionClass = classInfo.LookupEvaluationCode(evaluationCode)
		if data.SessionClass == nil {
			if rc.request.FormValue("submit") != "" {
				data.Invalid["evaluationCode"] = "Class not found"
			}
			return rc.respond(svc.templates.Eval1, http.StatusOK, &data)
		}
		data.EvaluateSession = true
		data.EvaluateConference = data.SessionClass.Session == model.NumSession-1
	}

	if rc.request.Method != "POST" {
		// Fill form from database.

		var (
			g                    errgroup.Group
			sessionEvaluation    *model.SessionEvaluation
			conferenceEvaluation *model.ConferenceEvaluation
			isInstructor         bool
		)

		if data.EvaluateSession {

			g.Go(func() error {
				var err error
				sessionEvaluation, err = svc.store.GetSessionEvaluation(rc.ctx, rc.participantID, data.SessionClass.Session)
				if err == store.ErrNotFound {
					sessionEvaluation = nil
					err = nil
				}
				return err
			})

			g.Go(func() error {
				// Determine if participant is isntructor for the class.
				participant, err := svc.store.GetParticipant(rc.ctx, rc.participantID)
				if err != nil {
					return err
				}
				sessionClass := classInfo.ParticipantSessionClasses(participant)[data.SessionClass.Session]
				isInstructor = sessionClass.Instructor && sessionClass.Class == data.SessionClass.Class
				return nil
			})
		}

		if data.EvaluateConference {
			g.Go(func() error {
				var err error
				conferenceEvaluation, err = svc.store.GetConferenceEvaluation(rc.ctx, rc.participantID)
				if err == store.ErrNotFound {
					conferenceEvaluation = nil
					err = nil
				}
				return err
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}

		data.Form.Set("isInstructor", blankOrYes(isInstructor))

		if sessionEvaluation == nil || sessionEvaluation.ClassNumber != data.SessionClass.Number {
			sessionEvaluation = &model.SessionEvaluation{}
		}
		setSessionEvaluationForm(data.Form, sessionEvaluation, "")
		data.Form.Set("hash", sessionEvaluation.HashEditFields())

		if data.EvaluateConference {
			if conferenceEvaluation == nil {
				conferenceEvaluation = &model.ConferenceEvaluation{}
			}
			setConferenceEvaluationForm(data.Form, conferenceEvaluation)
			data.Form.Set("chash", conferenceEvaluation.HashEditFields())
		}

		return rc.respond(svc.templates.Eval2, http.StatusOK, &data)
	}

	getRating := func(name string, required bool) int {
		n, _ := strconv.Atoi(rc.request.FormValue(name))
		if required && (n < 1 || n > 4) {
			data.Invalid[name] = "invalid"
		}
		return n
	}

	var sessionEvaluation *model.SessionEvaluation
	if data.EvaluateSession {
		sessionEvaluation = &model.SessionEvaluation{
			ParticipantID: rc.participantID,
			Session:       data.SessionClass.Session,
			ClassNumber:   data.SessionClass.Number,
			Updated:       time.Now().In(model.TimeLocation),
			Source:        "participant",
			Comments:      strings.TrimSpace(rc.request.FormValue("comments")),
		}
		if data.Form.Get("isInstructor") == "" {
			sessionEvaluation.KnowledgeRating = getRating("knowledge", true)
			sessionEvaluation.PresentationRating = getRating("presentation", true)
			sessionEvaluation.UsefulnessRating = getRating("usefulness", true)
			sessionEvaluation.OverallRating = getRating("overall", true)
		}
	}

	var conferenceEvaluation *model.ConferenceEvaluation
	if data.EvaluateConference {
		conferenceEvaluation = &model.ConferenceEvaluation{
			ParticipantID:           rc.participantID,
			Updated:                 time.Now().In(model.TimeLocation),
			Source:                  "participant",
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

	var description []string
	var g errgroup.Group

	if sessionEvaluation != nil {
		description = append(description, fmt.Sprintf("session %d", sessionEvaluation.Session+1))
		g.Go(func() error {
			return svc.store.SetSessionEvaluations(rc.ctx, []*model.SessionEvaluation{sessionEvaluation})
		})
	}

	if conferenceEvaluation != nil {
		description = append(description, "the conference")
		g.Go(func() error { return svc.store.SetConferenceEvaluation(rc.ctx, conferenceEvaluation) })
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return rc.redirect("/", "info", "Evaluation recorded for %s.", strings.Join(description, " and "))
}

func ratingString(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func setSessionEvaluationForm(form url.Values, e *model.SessionEvaluation, suffix string) {
	form.Set("knowledge"+suffix, ratingString(e.KnowledgeRating))
	form.Set("presentation"+suffix, ratingString(e.PresentationRating))
	form.Set("usefulness"+suffix, ratingString(e.UsefulnessRating))
	form.Set("overall"+suffix, ratingString(e.OverallRating))
	form.Set("comments"+suffix, e.Comments)
}

func setConferenceEvaluationForm(form url.Values, e *model.ConferenceEvaluation) {
	form.Set("experience", ratingString(e.ExperienceRating))
	form.Set("promotion", ratingString(e.PromotionRating))
	form.Set("registration", ratingString(e.RegistrationRating))
	form.Set("checkin", ratingString(e.CheckinRating))
	form.Set("midway", ratingString(e.MidwayRating))
	form.Set("lunch", ratingString(e.LunchRating))
	form.Set("facilities", ratingString(e.FacilitiesRating))
	form.Set("website", ratingString(e.WebsiteRating))
	form.Set("signageWayfinding", ratingString(e.SignageWayfindingRating))
	form.Set("learnTopics", e.LearnTopics)
	form.Set("teachTopics", e.TeachTopics)
	form.Set("comments", e.Comments)
}
