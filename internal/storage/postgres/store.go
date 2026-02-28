package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lutefd/baseline-api/internal/domain/opponents"
	"github.com/lutefd/baseline-api/internal/domain/sessions"
	"github.com/lutefd/baseline-api/internal/domain/stats"
	"github.com/lutefd/baseline-api/internal/domain/sync"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) EnsureDefaultUser(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, email)
		VALUES ($1, NULL)
		ON CONFLICT (id) DO NOTHING
	`, id)
	return err
}

func (s *Store) CreateSession(ctx context.Context, v sessions.Session) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (
			id, user_id, opponent_id, session_name, session_type, date, duration_minutes,
			rushed_shots, unforced_errors, long_rallies, direction_changes, composure,
			focus_text, followed_focus, is_match_win, notes, created_at, updated_at, deleted_at
		)
		VALUES (
			$1,$2,$3,$4,$5,$6,$7,
			$8,$9,$10,$11,$12,
			$13,$14,$15,$16,$17,$18,$19
		)
	`,
		v.ID, v.UserID, v.OpponentID, v.SessionName, v.SessionType, v.Date, v.DurationMinutes,
		v.RushedShots, v.UnforcedErrors, v.LongRallies, v.DirectionChanges, v.Composure,
		v.FocusText, v.FollowedFocus, v.IsMatchWin, v.Notes, v.CreatedAt, v.UpdatedAt, v.DeletedAt,
	)
	return err
}

func (s *Store) ListSessionsByUser(ctx context.Context, userID uuid.UUID, includeDeleted bool, limit int) ([]sessions.Session, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	query := `
		SELECT id, user_id, opponent_id, session_name, session_type, date, duration_minutes,
		       rushed_shots, unforced_errors, long_rallies, direction_changes, composure,
		       focus_text, followed_focus, is_match_win, notes, created_at, updated_at, deleted_at
		FROM sessions
		WHERE user_id = $1`
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}
	query += ` ORDER BY date DESC LIMIT $2`

	rows, err := s.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]sessions.Session, 0)
	for rows.Next() {
		var v sessions.Session
		if err := rows.Scan(
			&v.ID, &v.UserID, &v.OpponentID, &v.SessionName, &v.SessionType, &v.Date, &v.DurationMinutes,
			&v.RushedShots, &v.UnforcedErrors, &v.LongRallies, &v.DirectionChanges, &v.Composure,
			&v.FocusText, &v.FollowedFocus, &v.IsMatchWin, &v.Notes, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (s *Store) ListSessionsByDateRange(ctx context.Context, userID uuid.UUID, from, to *time.Time) ([]sessions.Session, error) {
	query := `
		SELECT id, user_id, opponent_id, session_name, session_type, date, duration_minutes,
		       rushed_shots, unforced_errors, long_rallies, direction_changes, composure,
		       focus_text, followed_focus, is_match_win, notes, created_at, updated_at, deleted_at
		FROM sessions
		WHERE user_id = $1 AND deleted_at IS NULL`
	args := []any{userID}
	if from != nil {
		query += fmt.Sprintf(" AND date >= $%d", len(args)+1)
		args = append(args, *from)
	}
	if to != nil {
		query += fmt.Sprintf(" AND date <= $%d", len(args)+1)
		args = append(args, *to)
	}
	query += ` ORDER BY date ASC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]sessions.Session, 0)
	for rows.Next() {
		var v sessions.Session
		if err := rows.Scan(
			&v.ID, &v.UserID, &v.OpponentID, &v.SessionName, &v.SessionType, &v.Date, &v.DurationMinutes,
			&v.RushedShots, &v.UnforcedErrors, &v.LongRallies, &v.DirectionChanges, &v.Composure,
			&v.FocusText, &v.FollowedFocus, &v.IsMatchWin, &v.Notes, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (s *Store) ListMatchSessionsByOpponent(ctx context.Context, userID, opponentID uuid.UUID) ([]sessions.Session, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, opponent_id, session_name, session_type, date, duration_minutes,
		       rushed_shots, unforced_errors, long_rallies, direction_changes, composure,
		       focus_text, followed_focus, is_match_win, notes, created_at, updated_at, deleted_at
		FROM sessions
		WHERE user_id = $1
		  AND opponent_id = $2
		  AND session_type IN ('match', 'friendly')
		  AND deleted_at IS NULL
		ORDER BY date DESC
	`, userID, opponentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]sessions.Session, 0)
	for rows.Next() {
		var v sessions.Session
		if err := rows.Scan(
			&v.ID, &v.UserID, &v.OpponentID, &v.SessionName, &v.SessionType, &v.Date, &v.DurationMinutes,
			&v.RushedShots, &v.UnforcedErrors, &v.LongRallies, &v.DirectionChanges, &v.Composure,
			&v.FocusText, &v.FollowedFocus, &v.IsMatchWin, &v.Notes, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (s *Store) CreateOpponent(ctx context.Context, v opponents.Opponent) error {
	v = withIdentityKey(v)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO opponents (id, identity_key, user_id, name, dominant_hand, play_style, notes, created_at, updated_at, deleted_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, v.ID, v.IdentityKey, v.UserID, v.Name, v.DominantHand, v.PlayStyle, v.Notes, v.CreatedAt, v.UpdatedAt, v.DeletedAt)
	return err
}

func (s *Store) ListOpponentsByUser(ctx context.Context, userID uuid.UUID, includeDeleted bool) ([]opponents.Opponent, error) {
	query := `
		SELECT id, identity_key, user_id, name, dominant_hand, play_style, notes, created_at, updated_at, deleted_at
		FROM opponents WHERE user_id = $1`
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}
	query += ` ORDER BY lower(name) ASC`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]opponents.Opponent, 0)
	for rows.Next() {
		var v opponents.Opponent
		if err := rows.Scan(&v.ID, &v.IdentityKey, &v.UserID, &v.Name, &v.DominantHand, &v.PlayStyle, &v.Notes, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (s *Store) UpsertSessionByUpdatedAt(ctx context.Context, incoming sessions.Session) (sync.MergeDecision, error) {
	var storedUpdatedAt time.Time
	var storedDeletedAt *time.Time
	err := s.pool.QueryRow(ctx, `SELECT updated_at, deleted_at FROM sessions WHERE id = $1`, incoming.ID).Scan(&storedUpdatedAt, &storedDeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if err := s.CreateSession(ctx, incoming); err != nil {
				return sync.DecisionIgnore, err
			}
			return sync.DecisionInsert, nil
		}
		return sync.DecisionIgnore, err
	}

	decision := sync.ResolveByUpdatedAt(incoming.UpdatedAt, storedUpdatedAt, incoming.DeletedAt, storedDeletedAt)
	if decision != sync.DecisionUpdate {
		return decision, nil
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE sessions SET
			user_id = $2,
			opponent_id = $3,
			session_name = $4,
			session_type = $5,
			date = $6,
			duration_minutes = $7,
			rushed_shots = $8,
			unforced_errors = $9,
			long_rallies = $10,
			direction_changes = $11,
			composure = $12,
			focus_text = $13,
			followed_focus = $14,
			is_match_win = $15,
			notes = $16,
			created_at = $17,
			updated_at = $18,
			deleted_at = $19
		WHERE id = $1
	`,
		incoming.ID, incoming.UserID, incoming.OpponentID, incoming.SessionName, incoming.SessionType, incoming.Date,
		incoming.DurationMinutes, incoming.RushedShots, incoming.UnforcedErrors, incoming.LongRallies,
		incoming.DirectionChanges, incoming.Composure, incoming.FocusText, incoming.FollowedFocus, incoming.IsMatchWin,
		incoming.Notes, incoming.CreatedAt, incoming.UpdatedAt, incoming.DeletedAt,
	)
	return decision, err
}

func (s *Store) UpsertOpponentByUpdatedAt(ctx context.Context, incoming opponents.Opponent) (sync.MergeDecision, error) {
	incoming = withIdentityKey(incoming)
	var storedUpdatedAt time.Time
	var storedDeletedAt *time.Time
	err := s.pool.QueryRow(ctx, `SELECT updated_at, deleted_at FROM opponents WHERE id = $1`, incoming.ID).Scan(&storedUpdatedAt, &storedDeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if err := s.CreateOpponent(ctx, incoming); err != nil {
				return sync.DecisionIgnore, err
			}
			return sync.DecisionInsert, nil
		}
		return sync.DecisionIgnore, err
	}

	decision := sync.ResolveByUpdatedAt(incoming.UpdatedAt, storedUpdatedAt, incoming.DeletedAt, storedDeletedAt)
	if decision != sync.DecisionUpdate {
		return decision, nil
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE opponents SET
			user_id = $2,
			identity_key = $3,
			name = $4,
			dominant_hand = $5,
			play_style = $6,
			notes = $7,
			created_at = $8,
			updated_at = $9,
			deleted_at = $10
		WHERE id = $1
	`, incoming.ID, incoming.UserID, incoming.IdentityKey, incoming.Name, incoming.DominantHand, incoming.PlayStyle, incoming.Notes,
		incoming.CreatedAt, incoming.UpdatedAt, incoming.DeletedAt)
	return decision, err
}

func (s *Store) UpsertMatchSetByUpdatedAt(ctx context.Context, incoming sessions.MatchSet) (sync.MergeDecision, error) {
	var storedUpdatedAt time.Time
	var storedDeletedAt *time.Time
	err := s.pool.QueryRow(ctx, `SELECT updated_at, deleted_at FROM match_sets WHERE id = $1`, incoming.ID).Scan(&storedUpdatedAt, &storedDeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_, err := s.pool.Exec(ctx, `
				INSERT INTO match_sets (id, session_id, set_number, player_games, opponent_games, created_at, updated_at, deleted_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			`, incoming.ID, incoming.SessionID, incoming.SetNumber, incoming.PlayerGames, incoming.OpponentGames,
				incoming.CreatedAt, incoming.UpdatedAt, incoming.DeletedAt)
			if err != nil {
				return sync.DecisionIgnore, err
			}
			return sync.DecisionInsert, nil
		}
		return sync.DecisionIgnore, err
	}

	decision := sync.ResolveByUpdatedAt(incoming.UpdatedAt, storedUpdatedAt, incoming.DeletedAt, storedDeletedAt)
	if decision != sync.DecisionUpdate {
		return decision, nil
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE match_sets
		SET session_id=$2, set_number=$3, player_games=$4, opponent_games=$5, created_at=$6, updated_at=$7, deleted_at=$8
		WHERE id = $1
	`, incoming.ID, incoming.SessionID, incoming.SetNumber, incoming.PlayerGames, incoming.OpponentGames,
		incoming.CreatedAt, incoming.UpdatedAt, incoming.DeletedAt)
	return decision, err
}

func (s *Store) PullChanges(ctx context.Context, userID uuid.UUID, updatedAfter time.Time) ([]sessions.Session, []sessions.MatchSet, []opponents.Opponent, error) {
	sessionsRows, err := s.pool.Query(ctx, `
		SELECT id, user_id, opponent_id, session_name, session_type, date, duration_minutes,
		       rushed_shots, unforced_errors, long_rallies, direction_changes, composure,
		       focus_text, followed_focus, is_match_win, notes, created_at, updated_at, deleted_at
		FROM sessions
		WHERE user_id = $1 AND updated_at > $2
		ORDER BY updated_at ASC
	`, userID, updatedAfter)
	if err != nil {
		return nil, nil, nil, err
	}
	defer sessionsRows.Close()

	sessionItems := make([]sessions.Session, 0)
	sessionIDs := make([]uuid.UUID, 0)
	for sessionsRows.Next() {
		var v sessions.Session
		if err := sessionsRows.Scan(
			&v.ID, &v.UserID, &v.OpponentID, &v.SessionName, &v.SessionType, &v.Date, &v.DurationMinutes,
			&v.RushedShots, &v.UnforcedErrors, &v.LongRallies, &v.DirectionChanges, &v.Composure,
			&v.FocusText, &v.FollowedFocus, &v.IsMatchWin, &v.Notes, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt,
		); err != nil {
			return nil, nil, nil, err
		}
		sessionItems = append(sessionItems, v)
		sessionIDs = append(sessionIDs, v.ID)
	}

	setItems := make([]sessions.MatchSet, 0)
	if len(sessionIDs) > 0 {
		setRows, err := s.pool.Query(ctx, `
			SELECT id, session_id, set_number, player_games, opponent_games, created_at, updated_at, deleted_at
			FROM match_sets
			WHERE session_id = ANY($1) AND updated_at > $2
			ORDER BY updated_at ASC
		`, sessionIDs, updatedAfter)
		if err != nil {
			return nil, nil, nil, err
		}
		defer setRows.Close()
		for setRows.Next() {
			var v sessions.MatchSet
			if err := setRows.Scan(&v.ID, &v.SessionID, &v.SetNumber, &v.PlayerGames, &v.OpponentGames, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt); err != nil {
				return nil, nil, nil, err
			}
			setItems = append(setItems, v)
		}
	}

	opponentRows, err := s.pool.Query(ctx, `
		SELECT id, identity_key, user_id, name, dominant_hand, play_style, notes, created_at, updated_at, deleted_at
		FROM opponents
		WHERE user_id = $1 AND updated_at > $2
		ORDER BY updated_at ASC
	`, userID, updatedAfter)
	if err != nil {
		return nil, nil, nil, err
	}
	defer opponentRows.Close()

	opponentItems := make([]opponents.Opponent, 0)
	for opponentRows.Next() {
		var v opponents.Opponent
		if err := opponentRows.Scan(&v.ID, &v.IdentityKey, &v.UserID, &v.Name, &v.DominantHand, &v.PlayStyle, &v.Notes, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt); err != nil {
			return nil, nil, nil, err
		}
		opponentItems = append(opponentItems, v)
	}

	return sessionItems, setItems, opponentItems, nil
}

func (s *Store) UpsertUserStats(ctx context.Context, userID uuid.UUID, us stats.UserStats) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_stats (
			user_id, total_sessions, total_matches, win_rate, avg_composure, avg_rushing_index,
			avg_unforced_errors_per_min, improvement_slope_composure, improvement_slope_rushing, last_calculated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (user_id)
		DO UPDATE SET
			total_sessions = EXCLUDED.total_sessions,
			total_matches = EXCLUDED.total_matches,
			win_rate = EXCLUDED.win_rate,
			avg_composure = EXCLUDED.avg_composure,
			avg_rushing_index = EXCLUDED.avg_rushing_index,
			avg_unforced_errors_per_min = EXCLUDED.avg_unforced_errors_per_min,
			improvement_slope_composure = EXCLUDED.improvement_slope_composure,
			improvement_slope_rushing = EXCLUDED.improvement_slope_rushing,
			last_calculated_at = EXCLUDED.last_calculated_at
	`, userID, us.TotalSessions, us.TotalMatches, us.WinRate, us.AvgComposure, us.AvgRushingIndex,
		us.AvgUnforcedErrorsPerMin, us.ImprovementSlopeComposure, us.ImprovementSlopeRushing, us.LastCalculatedAt)
	return err
}

func (s *Store) UpsertOpponentStats(ctx context.Context, opponentID uuid.UUID, v stats.OpponentStats) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO opponent_stats (
			opponent_id, matches_played, win_rate, avg_composure, avg_rushing_index, avg_set_differential, last_calculated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (opponent_id)
		DO UPDATE SET
			matches_played = EXCLUDED.matches_played,
			win_rate = EXCLUDED.win_rate,
			avg_composure = EXCLUDED.avg_composure,
			avg_rushing_index = EXCLUDED.avg_rushing_index,
			avg_set_differential = EXCLUDED.avg_set_differential,
			last_calculated_at = EXCLUDED.last_calculated_at
	`, opponentID, v.MatchesPlayed, v.WinRate, v.AvgComposure, v.AvgRushingIndex, v.AvgSetDifferential, v.LastCalculatedAt)
	return err
}

func (s *Store) ReplaceWeeklyStats(ctx context.Context, userID uuid.UUID, rows []stats.WeeklyStats) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM weekly_stats WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := tx.Exec(ctx, `
			INSERT INTO weekly_stats (user_id, week_start_date, avg_composure, avg_rushing_index, win_rate, matches_played)
			VALUES ($1,$2,$3,$4,$5,$6)
		`, userID, row.WeekStartDate, row.AvgComposure, row.AvgRushingIndex, row.WinRate, row.MatchesPlayed); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) GetUserStats(ctx context.Context, userID uuid.UUID) (stats.UserStats, error) {
	var out stats.UserStats
	err := s.pool.QueryRow(ctx, `
		SELECT total_sessions, total_matches, win_rate, avg_composure, avg_rushing_index,
		       avg_unforced_errors_per_min, improvement_slope_composure, improvement_slope_rushing,
		       last_calculated_at
		FROM user_stats WHERE user_id = $1
	`, userID).Scan(
		&out.TotalSessions, &out.TotalMatches, &out.WinRate, &out.AvgComposure, &out.AvgRushingIndex,
		&out.AvgUnforcedErrorsPerMin, &out.ImprovementSlopeComposure, &out.ImprovementSlopeRushing,
		&out.LastCalculatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return stats.UserStats{}, nil
		}
		return stats.UserStats{}, err
	}
	return out, nil
}

func (s *Store) ListMatchSetsBySessionIDs(ctx context.Context, sessionIDs []uuid.UUID) (map[uuid.UUID][]sessions.MatchSet, error) {
	result := make(map[uuid.UUID][]sessions.MatchSet)
	if len(sessionIDs) == 0 {
		return result, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, session_id, set_number, player_games, opponent_games, created_at, updated_at, deleted_at
		FROM match_sets
		WHERE session_id = ANY($1)
	`, sessionIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var v sessions.MatchSet
		if err := rows.Scan(&v.ID, &v.SessionID, &v.SetNumber, &v.PlayerGames, &v.OpponentGames, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt); err != nil {
			return nil, err
		}
		result[v.SessionID] = append(result[v.SessionID], v)
	}
	return result, rows.Err()
}

func (s *Store) GetOpponentStats(ctx context.Context, opponentID uuid.UUID) (stats.OpponentStats, error) {
	var out stats.OpponentStats
	err := s.pool.QueryRow(ctx, `
		SELECT matches_played, win_rate, avg_composure, avg_rushing_index, avg_set_differential, last_calculated_at
		FROM opponent_stats WHERE opponent_id = $1
	`, opponentID).Scan(
		&out.MatchesPlayed, &out.WinRate, &out.AvgComposure, &out.AvgRushingIndex, &out.AvgSetDifferential, &out.LastCalculatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return stats.OpponentStats{}, nil
		}
		return stats.OpponentStats{}, err
	}
	return out, nil
}

func withIdentityKey(in opponents.Opponent) opponents.Opponent {
	if strings.TrimSpace(in.IdentityKey) == "" {
		in.IdentityKey = in.ID.String()
	}
	return in
}
