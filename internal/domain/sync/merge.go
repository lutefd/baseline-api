package sync

import "time"

type MergeDecision string

const (
	DecisionInsert MergeDecision = "insert"
	DecisionUpdate MergeDecision = "update"
	DecisionIgnore MergeDecision = "ignore"
)

func ResolveByUpdatedAt(incomingUpdatedAt, storedUpdatedAt time.Time, incomingDeletedAt, storedDeletedAt *time.Time) MergeDecision {
	if storedUpdatedAt.IsZero() {
		return DecisionInsert
	}
	if !incomingUpdatedAt.After(storedUpdatedAt) {
		return DecisionIgnore
	}

	if incomingDeletedAt != nil {
		if storedDeletedAt == nil || incomingDeletedAt.After(*storedDeletedAt) || incomingDeletedAt.Equal(*storedDeletedAt) {
			return DecisionUpdate
		}
		return DecisionIgnore
	}

	return DecisionUpdate
}
