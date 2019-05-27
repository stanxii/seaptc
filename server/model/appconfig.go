package model

//go:generate go run gogen.go -input appconfig.go -output gen_appconfig.go AppConfig

// AppConfig is loaded once at application startup.
type AppConfig struct {
	XSRFKey  string   `json:"xsrfKey" datastore:"xsrfKey,noindex"`
	HMACKeys []string `json:"hmacKeys" datastore:"hmacKeys,noindex"`
	AdminIDs []string `json:"adminIDs" datastore:"adminIDs,noindex"`
	StaffIDs []string `json:"staffIDs" datastore:"staffIDs,noindex"`

	// Google Open ID for login
	LoginClient struct {
		ID     string `json:"id" datastore:"id,noindex"`
		Secret string `json:"secret" datastore:"secret,noindex"`
	} `json:"loginClient" datastore:"loginClient,noindex"`

	// Planning spreadsheet
	ClassesSheetURL                string `json:"classesSheetURL" datastore:"classesSheetURL,noindex,omitempty"`
	SuggestedSchedulesSheetURL     string `json:"suggestedScheduleSheetURL" datastore:"suggestedScheduleSheetURL,noindex,omitempty"`
	PlanningSheetServiceAccountKey string `json:"planningSheetServiceAccountKey" datastore:"planningSheetServiceAccountKey,noindex"`

	Junk1 string `json:"-" datastore:"year,noindex,omitempty"`
	Junk2 string `json:"-" datastore:"planningSheetKey,noindex,omitempty"`
	Junk3 string `json:"-" datastore:"classesURL,noindex,omitempty"`
	Junk4 string `json:"-" datastore:"suggestedScheduleURL,noindex,omitempty"`
	Junk5 string `json:"-" datastore:"planningSheetURL,noindex,omitempty"`
}
