package events

import (
	"api/internal/messaging"
	"api/internal/models"
	"api/internal/notifier"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

const UserInvitationName = "UserInvitation"
const UserInvitationPayloadName = "UserInvitationPayload"

type UserInvitationPayload struct {
	Type            string
	To              string
	From            string
	WebUrl          string
	BucketName      string
	Role            string
	RoleDescription string
	InviteUrl       string
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
	role string,
	inviteId string,
	webUrl string,
) UserInvitation {
	// Generate role descriptions
	roleDescription := getRoleDescription(role)

	// Create invite URL pointing to the invitation page
	inviteUrl := fmt.Sprintf("%s/invites/%s", webUrl, inviteId)

	return UserInvitation{
		Publisher: publisher,
		Payload: UserInvitationPayload{
			Type:            UserInvitationName,
			To:              to,
			From:            from,
			WebUrl:          webUrl,
			BucketName:      bucket.Name,
			Role:            role,
			RoleDescription: roleDescription,
			InviteUrl:       inviteUrl,
		},
	}
}

func getRoleDescription(role string) string {
	switch role {
	case "owner":
		return "manage all aspects of this bucket, including adding/removing users and files"
	case "contributor":
		return "upload, edit, and delete files in this bucket"
	case "viewer":
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

func (e *UserInvitation) callback(webUrl string, notifier notifier.INotifier) {
	subject := fmt.Sprintf("%s has invited you to SafeBucket", e.Payload.From)
	err := notifier.NotifyFromTemplate(e.Payload.To, subject, "user_invitation", e.Payload)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
	}
}
