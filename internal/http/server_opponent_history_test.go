package httpserver

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lutefd/baseline-api/internal/domain/sessions"
)

func TestBuildMatchHistory(t *testing.T) {
	win := true
	note := "good pressure in final set"
	sessionID := uuid.New()
	date := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)

	items := []sessions.Session{
		{
			ID:              sessionID,
			Date:            date,
			SessionName:     "League Match",
			IsMatchWin:      &win,
			Composure:       8,
			RushedShots:     6,
			UnforcedErrors:  4,
			DurationMinutes: 50,
			Notes:           &note,
		},
	}
	sets := map[uuid.UUID][]sessions.MatchSet{
		sessionID: {
			{SessionID: sessionID, PlayerGames: 6, OpponentGames: 3},
			{SessionID: sessionID, PlayerGames: 4, OpponentGames: 6},
		},
	}

	history := buildMatchHistory(items, sets)
	if len(history) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(history))
	}
	row := history[0]
	if row.SessionID != sessionID {
		t.Fatalf("unexpected session id: %s", row.SessionID)
	}
	if row.Date != date {
		t.Fatalf("unexpected date: %s", row.Date)
	}
	if row.SetDifferential != 1 {
		t.Fatalf("expected set differential 1, got %d", row.SetDifferential)
	}
	if row.RushingIndex != 0.2 {
		t.Fatalf("expected rushing index 0.2, got %.4f", row.RushingIndex)
	}
}
