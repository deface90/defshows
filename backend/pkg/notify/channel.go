// Package notify provides notification delivery channels.
package notify

import "context"

// Message is a channel-agnostic notification payload. Target identifies the
// recipient in whatever form the channel needs (a Telegram chat id, an APNs
// device token); Data carries channel-specific extras (e.g. show_id for a
// push's deep link) that channels which don't understand them simply ignore.
type Message struct {
	Target string
	Title  string
	Body   string
	Data   map[string]string
}

// Channel delivers a notification message.
type Channel interface {
	Send(ctx context.Context, msg Message) error
}
