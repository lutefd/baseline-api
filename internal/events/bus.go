package events

import (
	"context"
	"sync"
)

type Event struct {
	Name    string
	Payload any
}

type Handler func(context.Context, Event) error

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewBus() *Bus {
	return &Bus{handlers: map[string][]Handler{}}
}

func (b *Bus) Subscribe(name string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = append(b.handlers[name], handler)
}

func (b *Bus) Publish(ctx context.Context, e Event) error {
	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers[e.Name]...)
	b.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, e); err != nil {
			return err
		}
	}
	return nil
}
