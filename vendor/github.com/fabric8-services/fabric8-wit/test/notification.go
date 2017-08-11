package test

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/notification"
)

// NotificationChannel is a simple Sender impl that records the notifications for later verification
type NotificationChannel struct {
	Messages []notification.Message
}

// Send records each sent message in Messages
func (s *NotificationChannel) Send(ctx context.Context, msg notification.Message) {
	s.Messages = append(s.Messages, msg)
}
