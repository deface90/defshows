// Package notify provides notification delivery channels.
package notify

import "context"

// Message is a channel-agnostic notification payload.
type Message struct {
	ChatID int64
	Text   string
}

// Channel delivers a notification message.
type Channel interface {
	Send(ctx context.Context, msg Message) error
}
