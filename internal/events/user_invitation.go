package events

import (
	"fmt"

	"github.com/safebucket/safebucket/internal/models"

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

func (p UserInvitationPayload) EventName() string { return p.Type }

type UserInvitation struct {
	Payload UserInvitationPayload
}

func NewUserInvitation(
	to string,
	from string,
	bucket models.Bucket,
	group models.Group,
	inviteID string,
	webURL string,
) UserInvitation {
	groupDescription := getGroupDescription(group)

	inviteURL := fmt.Sprintf("%s/invites/%s", webURL, inviteID)

	return UserInvitation{
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
