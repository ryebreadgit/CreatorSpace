package server

func getFilterList() map[string]string {
	return map[string]string{
		"all":         "",
		"video":       "video_type = 'video' OR video_type = '' OR video_type is NULL",
		"short":       "video_type = 'short'",
		"watched":     "",
		"notwatched":  "",
		"public":      "availability = 'available'",
		"live":        "availability = 'live' OR video_type = 'Twitch'",
		"notlive":     "availability != 'live' AND video_type != 'Twitch'",
		"twitch":      "video_type = 'Twitch'",
		"unlisted":    "availability = 'unlisted'",
		"private":     "availability = 'private' OR availability = 'unavailable'",
		"unavailable": "availability = 'unavailable' OR availability = 'private' OR availability = 'unlisted'",
	}
}

func getSortList() map[string]string {
	return map[string]string{
		"newest":     "published_at DESC, created_at DESC, id DESC",
		"oldest":     "published_at ASC, created_at ASC, id ASC",
		"mostviews":  "CAST(views AS int) DESC, created_at DESC, id DESC",
		"leastviews": "CAST(views AS int) ASC, created_at ASC, id ASC",
		"mostlikes":  "CAST(likes AS int) DESC, created_at DESC, id DESC",
		"leastlikes": "CAST(likes AS int) ASC, created_at ASC, id ASC",
		"dateadded":  "id DESC",
	}
}

func intersection(a, b []string) []string {
	m := make(map[string]bool)
	for _, item := range a {
		m[item] = true
	}

	var intersect []string
	for _, item := range b {
		if _, ok := m[item]; ok {
			intersect = append(intersect, item)
		}
	}
	return intersect
}

func difference(a, b []string) []string {
	m := make(map[string]bool)
	for _, item := range b {
		m[item] = true
	}

	var diff []string
	for _, item := range a {
		if _, ok := m[item]; !ok {
			diff = append(diff, item)
		}
	}
	return diff
}
