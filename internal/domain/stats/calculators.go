package stats

import (
	"math"
	"sort"
	"time"

	"github.com/lutefd/baseline-api/internal/domain/sessions"
)

func RushingIndex(s sessions.Session) float64 {
	if s.DurationMinutes <= 0 {
		return 0
	}
	return float64(s.RushedShots+s.UnforcedErrors) / float64(s.DurationMinutes)
}

func WinRate(matches []sessions.Session) float64 {
	if len(matches) == 0 {
		return 0
	}
	wins := 0
	for _, m := range matches {
		if m.IsMatchWin != nil && *m.IsMatchWin {
			wins++
		}
	}
	return float64(wins) / float64(len(matches))
}

func SetDifferential(sets []sessions.MatchSet) int {
	total := 0
	for _, set := range sets {
		if set.DeletedAt != nil {
			continue
		}
		total += set.PlayerGames - set.OpponentGames
	}
	return total
}

func ImprovementSlopeComposure(items []sessions.Session) float64 {
	if len(items) < 2 {
		return 0
	}
	x := make([]float64, 0, len(items))
	y := make([]float64, 0, len(items))

	sorted := append([]sessions.Session(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date.Before(sorted[j].Date) })
	baseline := sorted[0].Date

	for _, s := range sorted {
		x = append(x, s.Date.Sub(baseline).Hours()/24)
		y = append(y, float64(s.Composure))
	}
	return linearRegressionSlope(x, y)
}

func ImprovementSlopeRushing(items []sessions.Session) float64 {
	if len(items) < 2 {
		return 0
	}
	x := make([]float64, 0, len(items))
	y := make([]float64, 0, len(items))

	sorted := append([]sessions.Session(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date.Before(sorted[j].Date) })
	baseline := sorted[0].Date

	for _, s := range sorted {
		x = append(x, s.Date.Sub(baseline).Hours()/24)
		y = append(y, RushingIndex(s))
	}
	return linearRegressionSlope(x, y)
}

func AverageUnforcedErrorsPerMin(items []sessions.Session) float64 {
	var totalUE, totalDuration int
	for _, s := range items {
		totalUE += s.UnforcedErrors
		totalDuration += s.DurationMinutes
	}
	if totalDuration == 0 {
		return 0
	}
	return float64(totalUE) / float64(totalDuration)
}

func AverageComposure(items []sessions.Session) float64 {
	if len(items) == 0 {
		return 0
	}
	sum := 0
	for _, s := range items {
		sum += s.Composure
	}
	return float64(sum) / float64(len(items))
}

func AverageRushingIndex(items []sessions.Session) float64 {
	if len(items) == 0 {
		return 0
	}
	var sum float64
	for _, s := range items {
		sum += RushingIndex(s)
	}
	return sum / float64(len(items))
}

func WeekStart(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	offset := weekday - 1
	start := t.AddDate(0, 0, -offset)
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
}

func Round(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func linearRegressionSlope(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(x))
	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}
	denominator := (n * sumX2) - (sumX * sumX)
	if denominator == 0 {
		return 0
	}
	return ((n * sumXY) - (sumX * sumY)) / denominator
}
