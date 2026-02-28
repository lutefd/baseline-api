package events

import (
	"context"
	"errors"
	"testing"
)

func TestBusPublishCallsHandlersInOrder(t *testing.T) {
	bus := NewBus()
	calls := make([]int, 0, 2)

	bus.Subscribe("SessionCreated", func(_ context.Context, _ Event) error {
		calls = append(calls, 1)
		return nil
	})
	bus.Subscribe("SessionCreated", func(_ context.Context, _ Event) error {
		calls = append(calls, 2)
		return nil
	})

	if err := bus.Publish(context.Background(), Event{Name: "SessionCreated"}); err != nil {
		t.Fatalf("publish returned error: %v", err)
	}

	if len(calls) != 2 || calls[0] != 1 || calls[1] != 2 {
		t.Fatalf("unexpected handler call sequence: %+v", calls)
	}
}

func TestBusPublishStopsOnFirstError(t *testing.T) {
	bus := NewBus()
	var calledSecond bool
	expectedErr := errors.New("handler failed")

	bus.Subscribe("SessionUpdated", func(_ context.Context, _ Event) error {
		return expectedErr
	})
	bus.Subscribe("SessionUpdated", func(_ context.Context, _ Event) error {
		calledSecond = true
		return nil
	})

	err := bus.Publish(context.Background(), Event{Name: "SessionUpdated"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if calledSecond {
		t.Fatalf("expected second handler not to run")
	}
}
