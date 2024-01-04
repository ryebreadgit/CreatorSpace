package api

// Add apiAbout structure. Include git commit hash, build date, and additional information
type apiVersionStruct struct {
	CommitHash string `json:"commit_hash"`
	BuildDate  string `json:"build_date"`
	Uptime     string `json:"uptime"`
	AppVersion string `json:"app_version"`
	GoVersion  string `json:"go_version"`
}
