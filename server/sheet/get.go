package sheet

import (
	"context"
	"fmt"
	"io"

	"github.com/seaptc/server/model"
	"golang.org/x/oauth2/google"
	"golang.org/x/xerrors"
)

func getBody(ctx context.Context, config *model.AppConfig, url string) (io.ReadCloser, error) {
	jwtConfig, err := google.JWTConfigFromJSON([]byte(config.PlanningSheetServiceAccountKey), "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		return nil, xerrors.Errorf("error parsing planning sheet service account key: %w", err)
	}
	client := jwtConfig.Client(ctx)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("fetch sheet returned %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func GetClasses(ctx context.Context, config *model.AppConfig) ([]*model.Class, error) {
	r, err := getBody(ctx, config, config.ClassesSheetURL)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return parseClasses(r)
}

func GetSuggestedSchedules(ctx context.Context, config *model.AppConfig) ([]*model.SuggestedSchedule, error) {
	r, err := getBody(ctx, config, config.SuggestedSchedulesSheetURL)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return parseSuggestedSchedules(r)
}
