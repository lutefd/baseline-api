package stats

import "time"

type UserStats struct {
	TotalSessions             int       `json:"totalSessions"`
	TotalMatches              int       `json:"totalMatches"`
	WinRate                   float64   `json:"winRate"`
	AvgComposure              float64   `json:"avgComposure"`
	AvgRushingIndex           float64   `json:"avgRushingIndex"`
	AvgUnforcedErrorsPerMin   float64   `json:"avgUnforcedErrorsPerMin"`
	ImprovementSlopeComposure float64   `json:"improvementSlopeComposure"`
	ImprovementSlopeRushing   float64   `json:"improvementSlopeRushing"`
	LastCalculatedAt          time.Time `json:"lastCalculatedAt"`
}

type OpponentStats struct {
	MatchesPlayed      int       `json:"matchesPlayed"`
	WinRate            float64   `json:"winRate"`
	AvgComposure       float64   `json:"avgComposure"`
	AvgRushingIndex    float64   `json:"avgRushingIndex"`
	AvgSetDifferential float64   `json:"avgSetDifferential"`
	LastCalculatedAt   time.Time `json:"lastCalculatedAt"`
}

type WeeklyStats struct {
	WeekStartDate   time.Time `json:"weekStartDate"`
	AvgComposure    float64   `json:"avgComposure"`
	AvgRushingIndex float64   `json:"avgRushingIndex"`
	WinRate         float64   `json:"winRate"`
	MatchesPlayed   int       `json:"matchesPlayed"`
}
