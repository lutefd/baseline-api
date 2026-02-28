package projections

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lutefd/baseline-api/internal/domain/sessions"
	"github.com/lutefd/baseline-api/internal/domain/stats"
)

type projectionStoreMock struct {
	sessions      []sessions.Session
	setsBySession map[uuid.UUID][]sessions.MatchSet

	upsertedUserID      uuid.UUID
	upsertedUserStats   stats.UserStats
	upsertedOpponent    map[uuid.UUID]stats.OpponentStats
	replacedWeeklyUser  uuid.UUID
	replacedWeeklyStats []stats.WeeklyStats
}

func (m *projectionStoreMock) ListSessionsByUser(_ context.Context, _ uuid.UUID, _ bool, _ int) ([]sessions.Session, error) {
	return m.sessions, nil
}

func (m *projectionStoreMock) ListMatchSetsBySessionIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]sessions.MatchSet, error) {
	return m.setsBySession, nil
}

func (m *projectionStoreMock) UpsertUserStats(_ context.Context, userID uuid.UUID, us stats.UserStats) error {
	m.upsertedUserID = userID
	m.upsertedUserStats = us
	return nil
}

func (m *projectionStoreMock) UpsertOpponentStats(_ context.Context, opponentID uuid.UUID, v stats.OpponentStats) error {
	if m.upsertedOpponent == nil {
		m.upsertedOpponent = make(map[uuid.UUID]stats.OpponentStats)
	}
	m.upsertedOpponent[opponentID] = v
	return nil
}

func (m *projectionStoreMock) ReplaceWeeklyStats(_ context.Context, userID uuid.UUID, rows []stats.WeeklyStats) error {
	m.replacedWeeklyUser = userID
	m.replacedWeeklyStats = rows
	return nil
}

func TestRecomputeForUserComputesUserOpponentAndWeeklyStats(t *testing.T) {
	userID := uuid.New()
	opponentID := uuid.New()
	win := true
	loss := false
	base := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	s1 := sessions.Session{
		ID:              uuid.New(),
		UserID:          userID,
		OpponentID:      &opponentID,
		SessionType:     "match",
		Date:            base,
		DurationMinutes: 60,
		RushedShots:     8,
		UnforcedErrors:  4,
		Composure:       7,
		IsMatchWin:      &win,
	}
	s2 := sessions.Session{
		ID:              uuid.New(),
		UserID:          userID,
		OpponentID:      &opponentID,
		SessionType:     "match",
		Date:            base.AddDate(0, 0, 7),
		DurationMinutes: 50,
		RushedShots:     10,
		UnforcedErrors:  6,
		Composure:       5,
		IsMatchWin:      &loss,
	}
	s3 := sessions.Session{
		ID:              uuid.New(),
		UserID:          userID,
		SessionType:     "class",
		Date:            base.AddDate(0, 0, 14),
		DurationMinutes: 45,
		RushedShots:     4,
		UnforcedErrors:  3,
		Composure:       8,
	}

	mock := &projectionStoreMock{
		sessions: []sessions.Session{s1, s2, s3},
		setsBySession: map[uuid.UUID][]sessions.MatchSet{
			s1.ID: {{SessionID: s1.ID, PlayerGames: 6, OpponentGames: 4}},
			s2.ID: {{SessionID: s2.ID, PlayerGames: 3, OpponentGames: 6}},
		},
	}

	svc := NewService(mock)
	if err := svc.RecomputeForUser(context.Background(), userID); err != nil {
		t.Fatalf("recompute failed: %v", err)
	}

	if mock.upsertedUserID != userID {
		t.Fatalf("unexpected user ID for upsert")
	}
	if mock.upsertedUserStats.TotalSessions != 3 {
		t.Fatalf("expected total sessions 3, got %d", mock.upsertedUserStats.TotalSessions)
	}
	if mock.upsertedUserStats.TotalMatches != 2 {
		t.Fatalf("expected total matches 2, got %d", mock.upsertedUserStats.TotalMatches)
	}
	if mock.upsertedUserStats.WinRate != 0.5 {
		t.Fatalf("expected win rate 0.5, got %.4f", mock.upsertedUserStats.WinRate)
	}

	opponentStats, ok := mock.upsertedOpponent[opponentID]
	if !ok {
		t.Fatalf("expected opponent stats to be upserted")
	}
	if opponentStats.MatchesPlayed != 2 {
		t.Fatalf("expected opponent matches 2, got %d", opponentStats.MatchesPlayed)
	}
	if opponentStats.AvgSetDifferential != -0.5 {
		t.Fatalf("expected avg set differential -0.5, got %.4f", opponentStats.AvgSetDifferential)
	}

	if mock.replacedWeeklyUser != userID {
		t.Fatalf("expected weekly stats user id to match")
	}
	if len(mock.replacedWeeklyStats) != 3 {
		t.Fatalf("expected 3 weekly buckets, got %d", len(mock.replacedWeeklyStats))
	}
}
