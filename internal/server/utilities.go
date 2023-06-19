package server

func getFilterList() map[string]string {
	return map[string]string{
		"all":         "",
		"video":       "video_type = 'video' OR video_type = '' OR video_type is NULL",
		"short":       "video_type = 'short'",
		"public":      "availability = 'available'",
		"live":        "availability = 'live' OR video_type = 'Twitch'",
		"notlive":     "availability != 'live' AND video_type != 'Twitch'",
		"twitch":      "video_type = 'Twitch'",
		"unlisted":    "availability = 'unlisted'",
		"private":     "availability = 'private' OR availability = 'unavailable'",
		"unavailable": "availability = 'unavailable' OR availability = 'private' OR availability = 'unlisted'",
	}
}
