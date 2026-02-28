package stats

import (
	"testing"

	"github.com/lutefd/baseline-api/internal/domain/sessions"
)

func TestCorrelationComposureVsWin(t *testing.T) {
	win := true
	loss := false
	items := []sessions.Session{
		{Composure: 3, IsMatchWin: &loss},
		{Composure: 4, IsMatchWin: &loss},
		{Composure: 8, IsMatchWin: &win},
		{Composure: 9, IsMatchWin: &win},
	}
	corr := CorrelationComposureVsWin(items)
	if corr == nil || *corr <= 0 {
		t.Fatalf("expected positive correlation, got %v", corr)
	}
}

func TestCorrelationInsufficientSamples(t *testing.T) {
	if got := CorrelationRushingVsWin(nil); got != nil {
		t.Fatalf("expected nil correlation, got %v", *got)
	}
}
