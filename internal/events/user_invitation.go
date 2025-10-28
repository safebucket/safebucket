package events

import (
	"encoding/json"
	"fmt"

	"api/internal/messaging"
	"api/internal/models"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

const (
	UserInvitationName        = "UserInvitation"
	UserInvitationPayloadName = "UserInvitationPayload"
)

type UserInvitationPayload struct {
	Type             string
	To               string
	From             string
	WebURL           string
	BucketName       string
	Group            models.Group
	GroupDescription string
	InviteURL        string
}

type UserInvitation struct {
	Publisher messaging.IPublisher
	Payload   UserInvitationPayload
}

func NewUserInvitation(
	publisher messaging.IPublisher,
	to string,
	from string,
	bucket models.Bucket,
	group models.Group,
	inviteID string,
	webURL string,
) UserInvitation {
	// Generate role descriptions
	groupDescription := getGroupDescription(group)

	// Create invite URL pointing to the invitation page
	inviteURL := fmt.Sprintf("%s/invites/%s", webURL, inviteID)

	return UserInvitation{
		Publisher: publisher,
		Payload: UserInvitationPayload{
			Type:             UserInvitationName,
			To:               to,
			From:             from,
			WebURL:           webURL,
			BucketName:       bucket.Name,
			Group:            group,
			GroupDescription: groupDescription,
			InviteURL:        inviteURL,
		},
	}
}

func getGroupDescription(group models.Group) string {
	switch group {
	case models.GroupOwner:
		return "manage all aspects of this bucket, including adding/removing users and files"
	case models.GroupContributor:
		return "upload, edit, and delete files in this bucket"
	case models.GroupViewer:
		return "view and download files from this bucket"
	default:
		return "collaborate on this bucket"
	}
}

func (e *UserInvitation) Trigger() {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		zap.L().Error("Error marshalling event payload", zap.Error(err))
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("type", e.Payload.Type)
	err = e.Publisher.Publish(msg)
	if err != nil {
		zap.L().Error("failed to trigger event", zap.Error(err))
	}
}

func (e *UserInvitation) callback(params *EventParams) error {
	e.Payload.WebURL = params.WebURL
	subject := fmt.Sprintf("%s has invited you to SafeBucket", e.Payload.From)
	err := params.Notifier.NotifyFromTemplate(e.Payload.To, subject, "user_invitation", e.Payload)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
		return err
	}
	return nil
}
