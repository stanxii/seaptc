package data

// AppConfig is loaded once at application startup.
type AppConfig struct {
	XSRFKey  string   `json:"xsrfKey" firestore:"xsrfKey"`
	HMACKeys []string `json:"hmacKeys" firestore:"hmacKeys"`
	AdminIDs []string `json:"adminIDs" firestore:"adminIDs"`

	// Google Open ID for login
	LoginClient struct {
		ID     string `json:"id" firestore:"id"`
		Secret string `json:"secret" firestore:"secret"`
	} `json:"loginClient" firestore:"loginClient"`

	// Planning spreadsheet
	PlanningSheetURL               string `json:"planningSheetURL" firestore:"planningSheetURL"`
	PlanningSheetServiceAccountKey string `json:"planningSheetServiceAccountKey" firestore:"planningSheetServiceAccountKey"`
}

const AppConfigPath = "misc/appconfig"
