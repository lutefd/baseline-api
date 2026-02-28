package sync

import (
	"testing"
	"time"
)

func TestResolveByUpdatedAt(t *testing.T) {
	now := time.Now().UTC()
	older := now.Add(-time.Hour)
	tombstone := now.Add(time.Minute)

	tests := []struct {
		name            string
		incomingUpdated time.Time
		storedUpdated   time.Time
		incomingDeleted *time.Time
		storedDeleted   *time.Time
		want            MergeDecision
	}{
		{name: "insert new", incomingUpdated: now, storedUpdated: time.Time{}, want: DecisionInsert},
		{name: "ignore stale", incomingUpdated: older, storedUpdated: now, want: DecisionIgnore},
		{name: "ignore equal", incomingUpdated: now, storedUpdated: now, want: DecisionIgnore},
		{name: "update newer", incomingUpdated: now, storedUpdated: older, want: DecisionUpdate},
		{name: "update tombstone", incomingUpdated: now, storedUpdated: older, incomingDeleted: &tombstone, want: DecisionUpdate},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveByUpdatedAt(tc.incomingUpdated, tc.storedUpdated, tc.incomingDeleted, tc.storedDeleted)
			if got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}
