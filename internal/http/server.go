package httpserver

import (
	"context"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lutefd/baseline-api/internal/auth"
	"github.com/lutefd/baseline-api/internal/domain/opponents"
	"github.com/lutefd/baseline-api/internal/domain/sessions"
	domainstats "github.com/lutefd/baseline-api/internal/domain/stats"
	domainsync "github.com/lutefd/baseline-api/internal/domain/sync"
	"github.com/lutefd/baseline-api/internal/projections"
	"github.com/lutefd/baseline-api/internal/storage/postgres"
)

type Dependencies struct {
	Store         *postgres.Store
	APIToken      string
	DefaultUserID uuid.UUID
}

type Server struct {
	store       *postgres.Store
	projection  *projections.Service
	auth        auth.Middleware
	defaultUser uuid.UUID
}

func NewServer(deps Dependencies) *Server {
	return &Server{
		store:       deps.Store,
		projection:  projections.NewService(deps.Store),
		auth:        auth.NewMiddleware(deps.APIToken, deps.DefaultUserID),
		defaultUser: deps.DefaultUserID,
	}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /v1/sessions", s.handleCreateSession)
	mux.HandleFunc("GET /v1/sessions", s.handleListSessions)
	mux.HandleFunc("POST /v1/opponents", s.handleCreateOpponent)
	mux.HandleFunc("GET /v1/opponents", s.handleListOpponents)
	mux.HandleFunc("POST /v1/sync/push", s.handleSyncPush)
	mux.HandleFunc("GET /v1/sync/pull", s.handleSyncPull)
	mux.HandleFunc("GET /v1/stats/overview", s.handleOverview)
	mux.HandleFunc("GET /v1/analysis/overview", s.handleOverview)
	mux.HandleFunc("GET /v1/analysis/trends", s.handleTrends)
	mux.HandleFunc("GET /v1/analysis/correlations", s.handleCorrelations)
	mux.HandleFunc("GET /v1/analysis/opponents/", s.handleOpponentAnalysis)

	return s.auth.Guard(loggingMiddleware(mux))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload sessions.Session
	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if payload.ID == uuid.Nil {
		payload.ID = uuid.New()
	}
	now := time.Now().UTC()
	payload.UserID = userID
	if payload.CreatedAt.IsZero() {
		payload.CreatedAt = now
	}
	payload.UpdatedAt = now

	if err := s.store.EnsureDefaultUser(r.Context(), userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.store.CreateSession(r.Context(), payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.projection.RecomputeForUser(r.Context(), userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, payload)
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	includeDeleted := r.URL.Query().Get("includeDeleted") == "true"
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	items, err := s.store.ListSessionsByUser(r.Context(), userID, includeDeleted, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": items})
}

func (s *Server) handleCreateOpponent(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload opponents.Opponent
	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if payload.ID == uuid.Nil {
		payload.ID = uuid.New()
	}
	now := time.Now().UTC()
	payload.UserID = userID
	if payload.CreatedAt.IsZero() {
		payload.CreatedAt = now
	}
	payload.UpdatedAt = now

	if err := s.store.EnsureDefaultUser(r.Context(), userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.store.CreateOpponent(r.Context(), payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, payload)
}

func (s *Server) handleListOpponents(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	includeDeleted := r.URL.Query().Get("includeDeleted") == "true"
	items, err := s.store.ListOpponentsByUser(r.Context(), userID, includeDeleted)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"opponents": items})
}

func (s *Server) handleSyncPush(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload domainsync.PushRequest
	if err := decodeJSON(r, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.store.EnsureDefaultUser(r.Context(), userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := domainsync.PushResponse{}

	for _, item := range payload.Opponents {
		item.UserID = userID
		decision, err := s.store.UpsertOpponentByUpdatedAt(r.Context(), item)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		applyCounts(&response.Opponents, decision)
	}

	for _, item := range payload.Sessions {
		item.UserID = userID
		decision, err := s.store.UpsertSessionByUpdatedAt(r.Context(), item)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		applyCounts(&response.Sessions, decision)
	}

	for _, item := range payload.MatchSets {
		decision, err := s.store.UpsertMatchSetByUpdatedAt(r.Context(), item)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		applyCounts(&response.MatchSets, decision)
	}

	response.ServerTimestamp = time.Now().UTC()
	if err := s.projection.RecomputeForUser(r.Context(), userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleSyncPull(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	updatedAfterRaw := r.URL.Query().Get("updatedAfter")
	if updatedAfterRaw == "" {
		http.Error(w, "updatedAfter is required", http.StatusBadRequest)
		return
	}
	updatedAfter, err := time.Parse(time.RFC3339, updatedAfterRaw)
	if err != nil {
		http.Error(w, "updatedAfter must be RFC3339", http.StatusBadRequest)
		return
	}

	sessionsChanged, matchSetsChanged, opponentsChanged, err := s.store.PullChanges(r.Context(), userID, updatedAfter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, domainsync.PullResponse{
		Sessions:  sessionsChanged,
		MatchSets: matchSetsChanged,
		Opponents: opponentsChanged,
	})
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	statsRow, err := s.store.GetUserStats(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	recent, err := s.store.ListSessionsByUser(r.Context(), userID, false, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"winRate":                   statsRow.WinRate,
		"avgComposure":              statsRow.AvgComposure,
		"avgRushingIndex":           statsRow.AvgRushingIndex,
		"improvementSlopeComposure": statsRow.ImprovementSlopeComposure,
		"improvementSlopeRushing":   statsRow.ImprovementSlopeRushing,
		"totalMatches":              statsRow.TotalMatches,
		"recentSessions":            recent,
	})
}

func (s *Server) handleOpponentAnalysis(w http.ResponseWriter, r *http.Request) {
	prefix := "/v1/analysis/opponents/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	rawID := strings.TrimPrefix(r.URL.Path, prefix)
	opponentID, err := uuid.Parse(rawID)
	if err != nil {
		http.Error(w, "invalid opponent id", http.StatusBadRequest)
		return
	}
	statsRow, err := s.store.GetOpponentStats(r.Context(), opponentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"matchesPlayed":      statsRow.MatchesPlayed,
		"winRate":            statsRow.WinRate,
		"avgComposure":       statsRow.AvgComposure,
		"avgRushingIndex":    statsRow.AvgRushingIndex,
		"avgSetDifferential": statsRow.AvgSetDifferential,
		"matchHistory":       []any{},
	})
}

func (s *Server) handleTrends(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	from, to, err := parseDateRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	items, err := s.store.ListSessionsByDateRange(r.Context(), userID, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	granularity := r.URL.Query().Get("granularity")
	if granularity == "" {
		granularity = "week"
	}
	series := aggregateTrends(items, granularity)
	writeJSON(w, http.StatusOK, map[string]any{
		"granularity": granularity,
		"series":      series,
	})
}

func (s *Server) handleCorrelations(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	from, to, err := parseDateRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	items, err := s.store.ListSessionsByDateRange(r.Context(), userID, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"composureVsWin":         domainstats.CorrelationComposureVsWin(items),
		"rushingVsWin":           domainstats.CorrelationRushingVsWin(items),
		"followedFocusVsRushing": domainstats.CorrelationFollowedFocusVsRushing(items),
		"longRalliesVsWin":       domainstats.CorrelationLongRalliesVsWin(items),
	})
}

func applyCounts(c *domainsync.EntityCounts, d domainsync.MergeDecision) {
	switch d {
	case domainsync.DecisionInsert:
		c.Inserted++
	case domainsync.DecisionUpdate:
		c.Updated++
	default:
		c.Ignored++
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s duration=%s", r.Method, r.URL.Path, time.Since(start))
	})
}

func userIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	return auth.UserIDFromContext(ctx)
}

func parseDateRange(r *http.Request) (*time.Time, *time.Time, error) {
	var from, to *time.Time
	if raw := r.URL.Query().Get("from"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, nil, err
		}
		from = &parsed
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, nil, err
		}
		to = &parsed
	}
	return from, to, nil
}

func aggregateTrends(items []sessions.Session, granularity string) []map[string]any {
	type bucket struct {
		sessions []sessions.Session
		matches  []sessions.Session
	}
	buckets := make(map[time.Time]*bucket)
	for _, item := range items {
		key := bucketStart(item.Date, granularity)
		if _, ok := buckets[key]; !ok {
			buckets[key] = &bucket{}
		}
		buckets[key].sessions = append(buckets[key].sessions, item)
		if item.IsMatch() {
			buckets[key].matches = append(buckets[key].matches, item)
		}
	}

	keys := make([]time.Time, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Before(keys[j]) })

	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		b := buckets[key]
		out = append(out, map[string]any{
			"bucketStartDate":  key,
			"avgComposure":     domainstats.Round(domainstats.AverageComposure(b.sessions)),
			"avgRushingIndex":  domainstats.Round(domainstats.AverageRushingIndex(b.sessions)),
			"winRate":          domainstats.Round(domainstats.WinRate(b.matches)),
			"matchesPlayed":    len(b.matches),
			"totalSessionRows": len(b.sessions),
		})
	}
	return out
}

func bucketStart(t time.Time, granularity string) time.Time {
	t = t.UTC()
	if granularity == "month" {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	return domainstats.WeekStart(t)
}
