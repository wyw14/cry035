package notifier

import (
	"context"
	"sync"
	"time"
)

type Message struct {
	Recipient string    `json:"recipient"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	SentAt    time.Time `json:"sent_at"`
}

type Local struct {
	mu       sync.RWMutex
	messages []Message
}

func NewLocal() *Local { return &Local{} }

func (n *Local) Send(ctx context.Context, message Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.messages = append(n.messages, message)
	return nil
}

func (n *Local) Messages() []Message {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]Message(nil), n.messages...)
}
