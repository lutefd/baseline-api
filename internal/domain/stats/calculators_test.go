package stats

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lutefd/baseline-api/internal/domain/sessions"
)

func TestRushingIndex(t *testing.T) {
	s := sessions.Session{RushedShots: 10, UnforcedErrors: 5, DurationMinutes: 30}
	if got := RushingIndex(s); got != 0.5 {
		t.Fatalf("expected 0.5, got %.4f", got)
	}
}

func TestWinRate(t *testing.T) {
	w := true
	l := false
	items := []sessions.Session{{IsMatchWin: &w}, {IsMatchWin: &l}, {IsMatchWin: &w}}
	if got := WinRate(items); got != 2.0/3.0 {
		t.Fatalf("unexpected win rate: %.6f", got)
	}
}

func TestSetDifferentialSkipsDeleted(t *testing.T) {
	now := time.Now()
	sets := []sessions.MatchSet{
		{PlayerGames: 6, OpponentGames: 4},
		{PlayerGames: 3, OpponentGames: 6, DeletedAt: &now},
	}
	if got := SetDifferential(sets); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
}

func TestImprovementSlopeComposure(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	items := []sessions.Session{
		{ID: uuid.New(), Date: start, Composure: 4},
		{ID: uuid.New(), Date: start.AddDate(0, 0, 10), Composure: 6},
		{ID: uuid.New(), Date: start.AddDate(0, 0, 20), Composure: 8},
	}
	if got := ImprovementSlopeComposure(items); got <= 0 {
		t.Fatalf("expected positive slope, got %.4f", got)
	}
}
