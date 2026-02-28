package opponents

import (
	"time"

	"github.com/google/uuid"
)

type Opponent struct {
	ID           uuid.UUID  `json:"id"`
	IdentityKey  string     `json:"identityKey"`
	UserID       uuid.UUID  `json:"userId"`
	Name         string     `json:"name"`
	DominantHand *string    `json:"dominantHand,omitempty"`
	PlayStyle    *string    `json:"playStyle,omitempty"`
	Notes        *string    `json:"notes,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
}
