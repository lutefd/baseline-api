package projections

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/lutefd/baseline-api/internal/domain/sessions"
	"github.com/lutefd/baseline-api/internal/domain/stats"
)

type Store interface {
	ListSessionsByUser(ctx context.Context, userID uuid.UUID, includeDeleted bool, limit int) ([]sessions.Session, error)
	ListMatchSetsBySessionIDs(ctx context.Context, sessionIDs []uuid.UUID) (map[uuid.UUID][]sessions.MatchSet, error)
	UpsertUserStats(ctx context.Context, userID uuid.UUID, us stats.UserStats) error
	UpsertOpponentStats(ctx context.Context, opponentID uuid.UUID, v stats.OpponentStats) error
	ReplaceWeeklyStats(ctx context.Context, userID uuid.UUID, rows []stats.WeeklyStats) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) RecomputeForUser(ctx context.Context, userID uuid.UUID) error {
	allSessions, err := s.store.ListSessionsByUser(ctx, userID, false, 20000)
	if err != nil {
		return err
	}

	matchSessions := make([]sessions.Session, 0)
	sessionIDs := make([]uuid.UUID, 0, len(allSessions))
	for _, item := range allSessions {
		sessionIDs = append(sessionIDs, item.ID)
		if item.IsMatch() {
			matchSessions = append(matchSessions, item)
		}
	}

	us := stats.UserStats{
		TotalSessions:             len(allSessions),
		TotalMatches:              len(matchSessions),
		WinRate:                   stats.Round(stats.WinRate(matchSessions)),
		AvgComposure:              stats.Round(stats.AverageComposure(allSessions)),
		AvgRushingIndex:           stats.Round(stats.AverageRushingIndex(allSessions)),
		AvgUnforcedErrorsPerMin:   stats.Round(stats.AverageUnforcedErrorsPerMin(allSessions)),
		ImprovementSlopeComposure: stats.Round(stats.ImprovementSlopeComposure(allSessions)),
		ImprovementSlopeRushing:   stats.Round(stats.ImprovementSlopeRushing(allSessions)),
		LastCalculatedAt:          time.Now().UTC(),
	}
	if err := s.store.UpsertUserStats(ctx, userID, us); err != nil {
		return err
	}

	if err := s.recomputeOpponents(ctx, matchSessions); err != nil {
		return err
	}

	weekly := computeWeeklyStats(allSessions)
	if err := s.store.ReplaceWeeklyStats(ctx, userID, weekly); err != nil {
		return err
	}

	return nil
}

func (s *Service) recomputeOpponents(ctx context.Context, matchSessions []sessions.Session) error {
	byOpponent := make(map[uuid.UUID][]sessions.Session)
	sessionIDs := make([]uuid.UUID, 0)
	for _, item := range matchSessions {
		if item.OpponentID == nil {
			continue
		}
		opponentID := *item.OpponentID
		byOpponent[opponentID] = append(byOpponent[opponentID], item)
		sessionIDs = append(sessionIDs, item.ID)
	}

	setsBySession, err := s.store.ListMatchSetsBySessionIDs(ctx, sessionIDs)
	if err != nil {
		return err
	}

	for opponentID, sessionsForOpponent := range byOpponent {
		var setDiffTotal int
		for _, item := range sessionsForOpponent {
			setDiffTotal += stats.SetDifferential(setsBySession[item.ID])
		}

		os := stats.OpponentStats{
			MatchesPlayed:      len(sessionsForOpponent),
			WinRate:            stats.Round(stats.WinRate(sessionsForOpponent)),
			AvgComposure:       stats.Round(stats.AverageComposure(sessionsForOpponent)),
			AvgRushingIndex:    stats.Round(stats.AverageRushingIndex(sessionsForOpponent)),
			AvgSetDifferential: 0,
			LastCalculatedAt:   time.Now().UTC(),
		}
		if len(sessionsForOpponent) > 0 {
			os.AvgSetDifferential = stats.Round(float64(setDiffTotal) / float64(len(sessionsForOpponent)))
		}
		if err := s.store.UpsertOpponentStats(ctx, opponentID, os); err != nil {
			return err
		}
	}

	return nil
}

func computeWeeklyStats(items []sessions.Session) []stats.WeeklyStats {
	type weeklyAccumulator struct {
		sessions []sessions.Session
		matches  []sessions.Session
	}
	acc := make(map[time.Time]*weeklyAccumulator)

	for _, item := range items {
		week := stats.WeekStart(item.Date)
		if _, ok := acc[week]; !ok {
			acc[week] = &weeklyAccumulator{}
		}
		acc[week].sessions = append(acc[week].sessions, item)
		if item.IsMatch() {
			acc[week].matches = append(acc[week].matches, item)
		}
	}

	weeks := make([]time.Time, 0, len(acc))
	for week := range acc {
		weeks = append(weeks, week)
	}
	sort.Slice(weeks, func(i, j int) bool { return weeks[i].Before(weeks[j]) })

	rows := make([]stats.WeeklyStats, 0, len(weeks))
	for _, week := range weeks {
		bucket := acc[week]
		rows = append(rows, stats.WeeklyStats{
			WeekStartDate:   week,
			AvgComposure:    stats.Round(stats.AverageComposure(bucket.sessions)),
			AvgRushingIndex: stats.Round(stats.AverageRushingIndex(bucket.sessions)),
			WinRate:         stats.Round(stats.WinRate(bucket.matches)),
			MatchesPlayed:   len(bucket.matches),
		})
	}
	return rows
}
