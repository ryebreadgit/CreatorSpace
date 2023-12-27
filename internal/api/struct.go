package api

// Add apiAbout structure. Include git commit hash, build date, and additional information
type apiAboutStruct struct {
	CommitHash string `json:"commit_hash"`
	BuildDate  string `json:"build_date"`
	Uptime     string `json:"uptime"`
	Version    string `json:"version"`
	GoVersion  string `json:"go_version"`
}
