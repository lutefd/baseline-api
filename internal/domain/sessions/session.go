package sessions

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"userId"`
	OpponentID       *uuid.UUID `json:"opponentId,omitempty"`
	SessionName      string     `json:"sessionName"`
	SessionType      string     `json:"sessionType"`
	Date             time.Time  `json:"date"`
	DurationMinutes  int        `json:"durationMinutes"`
	RushedShots      int        `json:"rushedShots"`
	UnforcedErrors   int        `json:"unforcedErrors"`
	LongRallies      int        `json:"longRallies"`
	DirectionChanges int        `json:"directionChanges"`
	Composure        int        `json:"composure"`
	FocusText        *string    `json:"focusText,omitempty"`
	FollowedFocus    *string    `json:"followedFocus,omitempty"`
	IsMatchWin       *bool      `json:"isMatchWin,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	DeletedAt        *time.Time `json:"deletedAt,omitempty"`
}

type MatchSet struct {
	ID            uuid.UUID  `json:"id"`
	SessionID     uuid.UUID  `json:"sessionId"`
	SetNumber     int        `json:"setNumber"`
	PlayerGames   int        `json:"playerGames"`
	OpponentGames int        `json:"opponentGames"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
}

func (s Session) IsDeleted() bool {
	return s.DeletedAt != nil
}

func (s Session) IsMatch() bool {
	return s.SessionType == "match"
}
