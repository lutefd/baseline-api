package stats

import "math"

import "github.com/lutefd/baseline-api/internal/domain/sessions"

func CorrelationComposureVsWin(items []sessions.Session) *float64 {
	x := make([]float64, 0)
	y := make([]float64, 0)
	for _, s := range items {
		if s.IsMatchWin == nil {
			continue
		}
		x = append(x, float64(s.Composure))
		if *s.IsMatchWin {
			y = append(y, 1)
		} else {
			y = append(y, 0)
		}
	}
	return pearson(x, y)
}

func CorrelationRushingVsWin(items []sessions.Session) *float64 {
	x := make([]float64, 0)
	y := make([]float64, 0)
	for _, s := range items {
		if s.IsMatchWin == nil {
			continue
		}
		x = append(x, RushingIndex(s))
		if *s.IsMatchWin {
			y = append(y, 1)
		} else {
			y = append(y, 0)
		}
	}
	return pearson(x, y)
}

func CorrelationFollowedFocusVsRushing(items []sessions.Session) *float64 {
	x := make([]float64, 0)
	y := make([]float64, 0)
	for _, s := range items {
		if s.FollowedFocus == nil {
			continue
		}
		val, ok := followedFocusToNumeric(*s.FollowedFocus)
		if !ok {
			continue
		}
		x = append(x, val)
		y = append(y, RushingIndex(s))
	}
	return pearson(x, y)
}

func CorrelationLongRalliesVsWin(items []sessions.Session) *float64 {
	x := make([]float64, 0)
	y := make([]float64, 0)
	for _, s := range items {
		if s.IsMatchWin == nil {
			continue
		}
		x = append(x, float64(s.LongRallies))
		if *s.IsMatchWin {
			y = append(y, 1)
		} else {
			y = append(y, 0)
		}
	}
	return pearson(x, y)
}

func followedFocusToNumeric(v string) (float64, bool) {
	switch v {
	case "yes":
		return 1, true
	case "partial":
		return 0.5, true
	case "no":
		return 0, true
	default:
		return 0, false
	}
}

func pearson(x, y []float64) *float64 {
	if len(x) != len(y) || len(x) < 2 {
		return nil
	}
	meanX := mean(x)
	meanY := mean(y)
	var num, denX, denY float64
	for i := range x {
		dx := x[i] - meanX
		dy := y[i] - meanY
		num += dx * dy
		denX += dx * dx
		denY += dy * dy
	}
	if denX == 0 || denY == 0 {
		return nil
	}
	v := num / (math.Sqrt(denX) * math.Sqrt(denY))
	rounded := Round(v)
	return &rounded
}

func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	var sum float64
	for _, item := range v {
		sum += item
	}
	return sum / float64(len(v))
}
