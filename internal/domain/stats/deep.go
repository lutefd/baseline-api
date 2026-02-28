package stats

import (
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/lutefd/baseline-api/internal/domain/sessions"
)

type InsightMetric struct {
	Sessions        int     `json:"sessions"`
	Matches         int     `json:"matches"`
	AvgRushingIndex float64 `json:"avgRushingIndex"`
	AvgComposure    float64 `json:"avgComposure"`
	WinRate         float64 `json:"winRate"`
}

type BehaviorDrift struct {
	Class             InsightMetric `json:"class"`
	Match             InsightMetric `json:"match"`
	DeltaRushingIndex float64       `json:"deltaRushingIndex"`
	DeltaComposure    float64       `json:"deltaComposure"`
}

type ScalarTrendPoint struct {
	BucketStartDate time.Time `json:"bucketStartDate"`
	Value           float64   `json:"value"`
	Sessions        int       `json:"sessions"`
}

type SetDifferentialTrendPoint struct {
	BucketStartDate    time.Time `json:"bucketStartDate"`
	AvgSetDifferential float64   `json:"avgSetDifferential"`
	Matches            int       `json:"matches"`
}

type OpponentBehaviorShift struct {
	OpponentID         uuid.UUID `json:"opponentId"`
	OpponentName       string    `json:"opponentName"`
	Matches            int       `json:"matches"`
	AvgRushingIndex    float64   `json:"avgRushingIndex"`
	AvgComposure       float64   `json:"avgComposure"`
	AvgSetDifferential float64   `json:"avgSetDifferential"`
	RushingSlope       float64   `json:"rushingSlope"`
	ComposureSlope     float64   `json:"composureSlope"`
}

type WeeklyVolatilityPoint struct {
	WeekStartDate   time.Time `json:"weekStartDate"`
	ComposureStdDev float64   `json:"composureStdDev"`
	RushingStdDev   float64   `json:"rushingStdDev"`
	Sessions        int       `json:"sessions"`
}

type ClutchIndicator struct {
	ClutchMatches           int     `json:"clutchMatches"`
	ClutchMatchesWithResult int     `json:"clutchMatchesWithResult"`
	WinRate                 float64 `json:"winRate"`
}

type FatigueSignal struct {
	AdjacentDayPairs         int      `json:"adjacentDayPairs"`
	AvgNextDayRushingDelta   float64  `json:"avgNextDayRushingDelta"`
	AvgNextDayComposureDelta float64  `json:"avgNextDayComposureDelta"`
	WeekendPairs             int      `json:"weekendPairs"`
	SaturdayToSundayRushing  *float64 `json:"saturdayToSundayRushing,omitempty"`
}

type DeepInsights struct {
	Granularity                 string                      `json:"granularity"`
	MatchVsClassBehavioralDrift BehaviorDrift               `json:"matchVsClassBehavioralDrift"`
	ComposureThresholdAnalysis  map[string]InsightMetric    `json:"composureThresholdAnalysis"`
	FocusAdherenceImpact        map[string]InsightMetric    `json:"focusAdherenceImpact"`
	RallyDensityTrend           []ScalarTrendPoint          `json:"rallyDensityTrend"`
	DirectionChangesTrend       []ScalarTrendPoint          `json:"directionChangesTrend"`
	SetDifferentialTrend        []SetDifferentialTrendPoint `json:"setDifferentialTrend"`
	OpponentBehavioralShift     []OpponentBehaviorShift     `json:"opponentBehavioralShift"`
	WeeklyVolatility            []WeeklyVolatilityPoint     `json:"weeklyVolatility"`
	ClutchIndicator             ClutchIndicator             `json:"clutchIndicator"`
	FatigueSignal               FatigueSignal               `json:"fatigueSignal"`
}

func BuildDeepInsights(items []sessions.Session, setsBySession map[uuid.UUID][]sessions.MatchSet, opponentNames map[uuid.UUID]string, granularity string) DeepInsights {
	if granularity != "month" {
		granularity = "week"
	}

	classSessions := filterByType(items, "class")
	matchSessions := filterByType(items, "match")

	classMetric := buildInsightMetric(classSessions)
	matchMetric := buildInsightMetric(matchSessions)

	insights := DeepInsights{
		Granularity: granularity,
		MatchVsClassBehavioralDrift: BehaviorDrift{
			Class:             classMetric,
			Match:             matchMetric,
			DeltaRushingIndex: Round(matchMetric.AvgRushingIndex - classMetric.AvgRushingIndex),
			DeltaComposure:    Round(matchMetric.AvgComposure - classMetric.AvgComposure),
		},
		ComposureThresholdAnalysis: composureThresholdAnalysis(items),
		FocusAdherenceImpact:       focusAdherenceImpact(items),
		RallyDensityTrend: buildRateTrend(items, granularity, func(s sessions.Session) float64 {
			if s.DurationMinutes <= 0 {
				return 0
			}
			return float64(s.LongRallies) / float64(s.DurationMinutes)
		}),
		DirectionChangesTrend: buildRateTrend(items, granularity, func(s sessions.Session) float64 {
			if s.DurationMinutes <= 0 {
				return 0
			}
			return float64(s.DirectionChanges) / float64(s.DurationMinutes)
		}),
		SetDifferentialTrend:    setDifferentialTrend(matchSessions, setsBySession, granularity),
		OpponentBehavioralShift: opponentBehavioralShift(matchSessions, setsBySession, opponentNames),
		WeeklyVolatility:        weeklyVolatility(items),
		ClutchIndicator:         clutchIndicator(matchSessions, setsBySession),
		FatigueSignal:           fatigueSignal(items),
	}

	return insights
}

func buildInsightMetric(items []sessions.Session) InsightMetric {
	matchesWithResult := make([]sessions.Session, 0)
	for _, item := range items {
		if item.IsMatch() && item.IsMatchWin != nil {
			matchesWithResult = append(matchesWithResult, item)
		}
	}
	return InsightMetric{
		Sessions:        len(items),
		Matches:         len(matchesWithResult),
		AvgRushingIndex: Round(AverageRushingIndex(items)),
		AvgComposure:    Round(AverageComposure(items)),
		WinRate:         Round(WinRate(matchesWithResult)),
	}
}

func filterByType(items []sessions.Session, sessionType string) []sessions.Session {
	out := make([]sessions.Session, 0)
	for _, item := range items {
		if item.SessionType == sessionType {
			out = append(out, item)
		}
	}
	return out
}

func composureThresholdAnalysis(items []sessions.Session) map[string]InsightMetric {
	buckets := map[string][]sessions.Session{
		"low":  {},
		"mid":  {},
		"high": {},
	}
	for _, item := range items {
		switch {
		case item.Composure < 5:
			buckets["low"] = append(buckets["low"], item)
		case item.Composure <= 7:
			buckets["mid"] = append(buckets["mid"], item)
		default:
			buckets["high"] = append(buckets["high"], item)
		}
	}
	return map[string]InsightMetric{
		"low":  buildInsightMetric(buckets["low"]),
		"mid":  buildInsightMetric(buckets["mid"]),
		"high": buildInsightMetric(buckets["high"]),
	}
}

func focusAdherenceImpact(items []sessions.Session) map[string]InsightMetric {
	buckets := map[string][]sessions.Session{
		"yes":     {},
		"partial": {},
		"no":      {},
	}
	for _, item := range items {
		if item.FollowedFocus == nil {
			continue
		}
		if _, ok := buckets[*item.FollowedFocus]; ok {
			buckets[*item.FollowedFocus] = append(buckets[*item.FollowedFocus], item)
		}
	}
	return map[string]InsightMetric{
		"yes":     buildInsightMetric(buckets["yes"]),
		"partial": buildInsightMetric(buckets["partial"]),
		"no":      buildInsightMetric(buckets["no"]),
	}
}

func buildRateTrend(items []sessions.Session, granularity string, rateFn func(sessions.Session) float64) []ScalarTrendPoint {
	type bucket struct {
		totalRate float64
		sessions  int
	}
	acc := map[time.Time]*bucket{}
	for _, item := range items {
		key := trendBucketStart(item.Date, granularity)
		if _, ok := acc[key]; !ok {
			acc[key] = &bucket{}
		}
		acc[key].totalRate += rateFn(item)
		acc[key].sessions++
	}

	keys := sortedTimeKeys(acc)
	out := make([]ScalarTrendPoint, 0, len(keys))
	for _, key := range keys {
		b := acc[key]
		value := 0.0
		if b.sessions > 0 {
			value = Round(b.totalRate / float64(b.sessions))
		}
		out = append(out, ScalarTrendPoint{BucketStartDate: key, Value: value, Sessions: b.sessions})
	}
	return out
}

func setDifferentialTrend(matchSessions []sessions.Session, setsBySession map[uuid.UUID][]sessions.MatchSet, granularity string) []SetDifferentialTrendPoint {
	type bucket struct {
		totalDiff int
		matches   int
	}
	acc := map[time.Time]*bucket{}
	for _, item := range matchSessions {
		key := trendBucketStart(item.Date, granularity)
		if _, ok := acc[key]; !ok {
			acc[key] = &bucket{}
		}
		acc[key].totalDiff += SetDifferential(setsBySession[item.ID])
		acc[key].matches++
	}

	keys := sortedTimeKeys(acc)
	out := make([]SetDifferentialTrendPoint, 0, len(keys))
	for _, key := range keys {
		b := acc[key]
		value := 0.0
		if b.matches > 0 {
			value = Round(float64(b.totalDiff) / float64(b.matches))
		}
		out = append(out, SetDifferentialTrendPoint{BucketStartDate: key, AvgSetDifferential: value, Matches: b.matches})
	}
	return out
}

func opponentBehavioralShift(matchSessions []sessions.Session, setsBySession map[uuid.UUID][]sessions.MatchSet, opponentNames map[uuid.UUID]string) []OpponentBehaviorShift {
	byOpponent := map[uuid.UUID][]sessions.Session{}
	for _, item := range matchSessions {
		if item.OpponentID == nil {
			continue
		}
		byOpponent[*item.OpponentID] = append(byOpponent[*item.OpponentID], item)
	}

	out := make([]OpponentBehaviorShift, 0)
	for opponentID, items := range byOpponent {
		if len(items) < 2 {
			continue
		}
		totalDiff := 0
		for _, item := range items {
			totalDiff += SetDifferential(setsBySession[item.ID])
		}
		name := opponentNames[opponentID]
		if name == "" {
			name = opponentID.String()
		}
		avgSetDiff := 0.0
		if len(items) > 0 {
			avgSetDiff = Round(float64(totalDiff) / float64(len(items)))
		}

		out = append(out, OpponentBehaviorShift{
			OpponentID:         opponentID,
			OpponentName:       name,
			Matches:            len(items),
			AvgRushingIndex:    Round(AverageRushingIndex(items)),
			AvgComposure:       Round(AverageComposure(items)),
			AvgSetDifferential: avgSetDiff,
			RushingSlope:       Round(ImprovementSlopeRushing(items)),
			ComposureSlope:     Round(ImprovementSlopeComposure(items)),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Matches == out[j].Matches {
			return out[i].OpponentName < out[j].OpponentName
		}
		return out[i].Matches > out[j].Matches
	})
	return out
}

func weeklyVolatility(items []sessions.Session) []WeeklyVolatilityPoint {
	byWeek := map[time.Time][]sessions.Session{}
	for _, item := range items {
		week := WeekStart(item.Date)
		byWeek[week] = append(byWeek[week], item)
	}

	keys := sortedTimeKeys(byWeek)
	out := make([]WeeklyVolatilityPoint, 0, len(keys))
	for _, key := range keys {
		weekItems := byWeek[key]
		composureSeries := make([]float64, 0, len(weekItems))
		rushingSeries := make([]float64, 0, len(weekItems))
		for _, item := range weekItems {
			composureSeries = append(composureSeries, float64(item.Composure))
			rushingSeries = append(rushingSeries, RushingIndex(item))
		}
		out = append(out, WeeklyVolatilityPoint{
			WeekStartDate:   key,
			ComposureStdDev: Round(stdDev(composureSeries)),
			RushingStdDev:   Round(stdDev(rushingSeries)),
			Sessions:        len(weekItems),
		})
	}
	return out
}

func clutchIndicator(matchSessions []sessions.Session, setsBySession map[uuid.UUID][]sessions.MatchSet) ClutchIndicator {
	clutchMatches := 0
	withResult := 0
	wins := 0
	for _, item := range matchSessions {
		diff := SetDifferential(setsBySession[item.ID])
		if math.Abs(float64(diff)) > 2 {
			continue
		}
		clutchMatches++
		if item.IsMatchWin != nil {
			withResult++
			if *item.IsMatchWin {
				wins++
			}
		}
	}
	winRate := 0.0
	if withResult > 0 {
		winRate = Round(float64(wins) / float64(withResult))
	}
	return ClutchIndicator{
		ClutchMatches:           clutchMatches,
		ClutchMatchesWithResult: withResult,
		WinRate:                 winRate,
	}
}

func fatigueSignal(items []sessions.Session) FatigueSignal {
	if len(items) < 2 {
		return FatigueSignal{}
	}
	sorted := append([]sessions.Session(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date.Before(sorted[j].Date) })

	adjacentPairs := 0
	totalRushingDelta := 0.0
	totalComposureDelta := 0.0
	weekendPairs := 0
	totalWeekendRushingDelta := 0.0

	for i := 0; i < len(sorted)-1; i++ {
		curr := sorted[i]
		next := sorted[i+1]
		currDay := time.Date(curr.Date.UTC().Year(), curr.Date.UTC().Month(), curr.Date.UTC().Day(), 0, 0, 0, 0, time.UTC)
		nextDay := time.Date(next.Date.UTC().Year(), next.Date.UTC().Month(), next.Date.UTC().Day(), 0, 0, 0, 0, time.UTC)
		if int(nextDay.Sub(currDay).Hours()/24) != 1 {
			continue
		}
		adjacentPairs++
		rushingDelta := RushingIndex(next) - RushingIndex(curr)
		composureDelta := float64(next.Composure - curr.Composure)
		totalRushingDelta += rushingDelta
		totalComposureDelta += composureDelta

		if currDay.Weekday() == time.Saturday && nextDay.Weekday() == time.Sunday {
			weekendPairs++
			totalWeekendRushingDelta += rushingDelta
		}
	}

	out := FatigueSignal{AdjacentDayPairs: adjacentPairs, WeekendPairs: weekendPairs}
	if adjacentPairs > 0 {
		out.AvgNextDayRushingDelta = Round(totalRushingDelta / float64(adjacentPairs))
		out.AvgNextDayComposureDelta = Round(totalComposureDelta / float64(adjacentPairs))
	}
	if weekendPairs > 0 {
		value := Round(totalWeekendRushingDelta / float64(weekendPairs))
		out.SaturdayToSundayRushing = &value
	}
	return out
}

func stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, v := range values {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(values))
	return math.Sqrt(variance)
}

func trendBucketStart(t time.Time, granularity string) time.Time {
	t = t.UTC()
	if granularity == "month" {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	return WeekStart(t)
}

func sortedTimeKeys[T any](in map[time.Time]T) []time.Time {
	keys := make([]time.Time, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Before(keys[j]) })
	return keys
}
