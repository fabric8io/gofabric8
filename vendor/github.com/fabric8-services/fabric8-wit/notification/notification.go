package notification

import (
	"context"
	"net/http"
	"net/url"

	"fmt"

	"github.com/fabric8-services/fabric8-wit/goasupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/notification/client"
	goaclient "github.com/goadesign/goa/client"
	goauuid "github.com/goadesign/goa/uuid"
	uuid "github.com/satori/go.uuid"
)

// Channel is a simple interface between the notifying component and the notificaiton impl
type Channel interface {
	Send(context.Context, Message)
}

// Message represents a new event of a Type for a Target performed by a User
// See helper constructors like NewWorkItemCreated, NewCommentUpdated
type Message struct {
	MessageID   uuid.UUID // unique ID per event
	UserID      *string
	TargetID    string
	MessageType string
}

func (m Message) String() string {
	return fmt.Sprintf("id:%v type:%v by:%v for:%v", m.MessageID, m.MessageType, m.UserID, m.TargetID)
}

// NewWorkItemCreated creates a new message instance for the newly created WorkItemID
func NewWorkItemCreated(workitemID string) Message {
	return Message{MessageID: uuid.NewV4(), MessageType: "workitem.create", TargetID: workitemID}
}

// NewWorkItemUpdated creates a new message instance for the updated WorkItemID
func NewWorkItemUpdated(workitemID string) Message {
	return Message{MessageID: uuid.NewV4(), MessageType: "workitem.update", TargetID: workitemID}
}

// NewCommentCreated creates a new message instance for the newly created CommentID
func NewCommentCreated(commentID string) Message {
	return Message{MessageID: uuid.NewV4(), MessageType: "comment.create", TargetID: commentID}
}

// NewCommentUpdated creates a new message instance for the updated CommentID
func NewCommentUpdated(commentID string) Message {
	return Message{MessageID: uuid.NewV4(), MessageType: "comment.update", TargetID: commentID}
}

func setCurrentIdentity(ctx context.Context, msg *Message) {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		uID := currentUserIdentityID.String()
		msg.UserID = &uID
	}
}

// DevNullChannel is the default configured channel. It does nothing.
type DevNullChannel struct{}

// Send NO-OP
func (d *DevNullChannel) Send(context.Context, Message) {}

// ServiceConfiguration holds configuration options required to interact with the fabric8-notification API
type ServiceConfiguration interface {
	GetNotificationServiceURL() string
}

// Service is a simple client Channel to the fabric8-notification service
type Service struct {
	config ServiceConfiguration
}

func validateConfig(config ServiceConfiguration) error {
	_, err := url.Parse(config.GetNotificationServiceURL())
	if err != nil {
		return fmt.Errorf("Invalid NotificationServiceURL %v cause %v", config.GetNotificationServiceURL(), err.Error())
	}
	return nil
}

// NewServiceChannel sends notification messages to the fabric8-notification service
func NewServiceChannel(config ServiceConfiguration) (Channel, error) {
	err := validateConfig(config)
	if err != nil {
		return nil, err
	}
	return &Service{config: config}, nil
}

// Send invokes the fabric8-notification API
func (s *Service) Send(ctx context.Context, msg Message) {
	go func(ctx context.Context, msg Message) {
		setCurrentIdentity(ctx, &msg)

		u, err := url.Parse(s.config.GetNotificationServiceURL())
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"url": s.config.GetNotificationServiceURL(),
				"err": err,
			}, "unable to parse GetNotificationServiceURL")
		}

		cl := client.New(goaclient.HTTPClientDoer(http.DefaultClient))
		cl.Host = u.Host
		cl.Scheme = u.Scheme
		cl.SetJWTSigner(goasupport.NewForwardSigner(ctx))

		msgID := goauuid.UUID(msg.MessageID)

		resp, err := cl.SendNotify(
			ctx,
			client.SendNotifyPath(),
			&client.SendNotifyPayload{
				Data: &client.Notification{
					Type: "notifications",
					ID:   &msgID,
					Attributes: &client.NotificationAttributes{
						Type: msg.MessageType,
						ID:   msg.TargetID,
					},
				},
			},
		)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"message_id": msg.MessageID,
				"type":       msg.MessageType,
				"target_id":  msg.TargetID,
				"err":        err,
			}, "unable to send notification")
		} else if resp.StatusCode >= 400 {
			log.Error(ctx, map[string]interface{}{
				"status":     resp.StatusCode,
				"message_id": msg.MessageID,
				"type":       msg.MessageType,
				"target_id":  msg.TargetID,
				"err":        err,
			}, "unexpected response code")
		}
		defer resp.Body.Close()

	}(ctx, msg)
}
