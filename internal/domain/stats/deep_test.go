package stats

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lutefd/baseline-api/internal/domain/sessions"
)

func TestBuildDeepInsights(t *testing.T) {
	userID := uuid.New()
	opponentID := uuid.New()
	yes := "yes"
	partial := "partial"
	no := "no"
	win := true
	loss := false

	s1 := sessions.Session{
		ID:               uuid.New(),
		UserID:           userID,
		OpponentID:       &opponentID,
		SessionType:      "match",
		Date:             time.Date(2026, 2, 7, 9, 0, 0, 0, time.UTC),
		DurationMinutes:  60,
		RushedShots:      8,
		UnforcedErrors:   4,
		LongRallies:      20,
		DirectionChanges: 15,
		Composure:        8,
		FollowedFocus:    &yes,
		IsMatchWin:       &win,
	}
	s2 := sessions.Session{
		ID:               uuid.New(),
		UserID:           userID,
		OpponentID:       &opponentID,
		SessionType:      "match",
		Date:             time.Date(2026, 2, 8, 9, 0, 0, 0, time.UTC),
		DurationMinutes:  55,
		RushedShots:      12,
		UnforcedErrors:   7,
		LongRallies:      18,
		DirectionChanges: 11,
		Composure:        4,
		FollowedFocus:    &no,
		IsMatchWin:       &loss,
	}
	s3 := sessions.Session{
		ID:               uuid.New(),
		UserID:           userID,
		SessionType:      "class",
		Date:             time.Date(2026, 2, 15, 9, 0, 0, 0, time.UTC),
		DurationMinutes:  50,
		RushedShots:      6,
		UnforcedErrors:   3,
		LongRallies:      22,
		DirectionChanges: 14,
		Composure:        6,
		FollowedFocus:    &partial,
	}
	s4 := sessions.Session{
		ID:               uuid.New(),
		UserID:           userID,
		OpponentID:       &opponentID,
		SessionType:      "friendly",
		Date:             time.Date(2026, 2, 16, 9, 0, 0, 0, time.UTC),
		DurationMinutes:  45,
		RushedShots:      7,
		UnforcedErrors:   5,
		LongRallies:      16,
		DirectionChanges: 9,
		Composure:        7,
		FollowedFocus:    &yes,
		IsMatchWin:       &win,
	}

	insights := BuildDeepInsights([]sessions.Session{s1, s2, s3, s4}, map[uuid.UUID][]sessions.MatchSet{
		s1.ID: {{SessionID: s1.ID, PlayerGames: 6, OpponentGames: 4}},
		s2.ID: {{SessionID: s2.ID, PlayerGames: 4, OpponentGames: 6}},
		s4.ID: {{SessionID: s4.ID, PlayerGames: 6, OpponentGames: 3}},
	}, map[uuid.UUID]string{opponentID: "Rival"}, "week")

	if insights.MatchVsClassBehavioralDrift.Match.Sessions != 3 {
		t.Fatalf("expected 3 competitive sessions")
	}
	if insights.MatchVsClassBehavioralDrift.Class.Sessions != 1 {
		t.Fatalf("expected 1 class session")
	}
	if insights.ComposureThresholdAnalysis["low"].Sessions != 1 {
		t.Fatalf("expected low composure bucket to have 1 session")
	}
	if insights.FocusAdherenceImpact["yes"].Sessions != 2 {
		t.Fatalf("expected followedFocus yes bucket to have 2 sessions")
	}
	if len(insights.SetDifferentialTrend) == 0 {
		t.Fatalf("expected set differential trend")
	}
	if len(insights.OpponentBehavioralShift) != 1 {
		t.Fatalf("expected one recurrent opponent shift")
	}
	if insights.ClutchIndicator.ClutchMatches != 2 {
		t.Fatalf("expected two clutch matches, got %d", insights.ClutchIndicator.ClutchMatches)
	}
	if insights.FatigueSignal.AdjacentDayPairs < 1 {
		t.Fatalf("expected adjacent day fatigue pair")
	}
}
