package models

type RankingEntryResponse struct {
	UserID                  string  `json:"user_id"`
	TagID                   string  `json:"tag_id,omitempty"`
	Username                string  `json:"username,omitempty"`
	DisplayName             string  `json:"display_name,omitempty"`
	AvatarURL               string  `json:"avatar_url,omitempty"`
	TitleLabel              string  `json:"title_label,omitempty"`
	Score                   float64 `json:"score"`
	FollowersCount          int64   `json:"followers_count"`
	FollowersGained30d      int64   `json:"followers_gained_30d"`
	StarsReceivedTotal      int64   `json:"stars_received_total"`
	StarsReceived30d        int64   `json:"stars_received_30d"`
	ForksReceivedTotal      int64   `json:"forks_received_total"`
	ForksReceived30d        int64   `json:"forks_received_30d"`
	PublicRepositoriesCount int64   `json:"public_repositories_count"`
	ActivityPointsTotal     int64   `json:"activity_points_total"`
	ActivityPoints30d       int64   `json:"activity_points_30d"`
	ActiveWeeksLast8        int64   `json:"active_weeks_last_8"`
	ActiveMonthsCount       int64   `json:"active_months_count"`
	SubjectScore            float64 `json:"subject_score"`
}

type RankingResponse struct {
	Entries []RankingEntryResponse `json:"entries"`
	Total   uint64                 `json:"total"`
}
