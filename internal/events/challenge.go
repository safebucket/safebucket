package events

import (
	"api/internal/messaging"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

const ChallengeUserInviteName = "ChallengeUserInvite"
const ChallengeUserInvitePayloadName = "ChallengeUserInvitePayload"

type ChallengeUserInvitePayload struct {
	Type         string
	Secret       string
	To           string
	From         string
	WebUrl       string
	ChallengeUrl string
}

type ChallengeUserInvite struct {
	Publisher messaging.IPublisher
	Payload   ChallengeUserInvitePayload
}

func NewChallengeUserInvite(
	publisher messaging.IPublisher,
	secret string,
	to string,
	inviteID string,
	challengeID string,
	webUrl string,
) ChallengeUserInvite {
	challengeUrl := fmt.Sprintf("%s/invites/%s/challenges/%s", webUrl, inviteID, challengeID)
	return ChallengeUserInvite{
		Publisher: publisher,
		Payload: ChallengeUserInvitePayload{
			Type:         ChallengeUserInviteName,
			Secret:       secret,
			To:           to,
			WebUrl:       webUrl,
			ChallengeUrl: challengeUrl,
		},
	}
}

func (e *ChallengeUserInvite) Trigger() {
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

func (e *ChallengeUserInvite) callback(params *EventParams) error {
	e.Payload.WebUrl = params.WebUrl
	subject := fmt.Sprintf("%s has invited you", "")
	err := params.Notifier.NotifyFromTemplate(e.Payload.To, subject, "user_invited", e.Payload)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
		return err
	}
	return nil
}
