package sync

import (
	"time"

	"github.com/lutefd/baseline-api/internal/domain/opponents"
	"github.com/lutefd/baseline-api/internal/domain/sessions"
)

type PushRequest struct {
	Sessions  []sessions.Session   `json:"sessions"`
	MatchSets []sessions.MatchSet  `json:"matchSets"`
	Opponents []opponents.Opponent `json:"opponents"`
}

type EntityCounts struct {
	Inserted int `json:"inserted"`
	Updated  int `json:"updated"`
	Ignored  int `json:"ignored"`
}

type PushResponse struct {
	Sessions        EntityCounts `json:"sessions"`
	MatchSets       EntityCounts `json:"matchSets"`
	Opponents       EntityCounts `json:"opponents"`
	ServerTimestamp time.Time    `json:"serverTimestamp"`
}

type PullResponse struct {
	Sessions  []sessions.Session   `json:"sessions"`
	MatchSets []sessions.MatchSet  `json:"matchSets"`
	Opponents []opponents.Opponent `json:"opponents"`
}
