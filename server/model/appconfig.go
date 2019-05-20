package model

import "time"

//go:generate go run gogen.go -input appconfig.go -output gen_appconfig.go AppConfig

// AppConfig is loaded once at application startup.
type AppConfig struct {
	XSRFKey  string   `json:"xsrfKey" firestore:"xsrfKey"`
	HMACKeys []string `json:"hmacKeys" firestore:"hmacKeys"`
	AdminIDs []string `json:"adminIDs" firestore:"adminIDs"`
	StaffIDs []string `json:"staffIDs" firestore:"staffIDs"`

	// Google Open ID for login
	LoginClient struct {
		ID     string `json:"id" firestore:"id"`
		Secret string `json:"secret" firestore:"secret"`
	} `json:"loginClient" firestore:"loginClient"`

	// Planning spreadsheet
	PlanningSheetURL               string `json:"planningSheetURL" firestore:"planningSheetURL"`
	PlanningSheetServiceAccountKey string `json:"planningSheetServiceAccountKey" firestore:"planningSheetServiceAccountKey"`

	LastUpdateTime time.Time `json:"lastUpdateTime" firestore:"lastUpdateTime,serverTimestamp"`
}
