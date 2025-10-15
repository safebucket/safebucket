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

const PasswordResetChallengeName = "PasswordResetChallenge"
const PasswordResetChallengePayloadName = "PasswordResetChallengePayload"

type PasswordResetChallengePayload struct {
	Type         string
	Secret       string
	To           string
	WebUrl       string
	ChallengeUrl string
}

type PasswordResetChallengeEvent struct {
	Publisher messaging.IPublisher
	Payload   PasswordResetChallengePayload
}

func NewPasswordResetChallenge(
	publisher messaging.IPublisher,
	secret string,
	to string,
	challengeID string,
	webUrl string,
) PasswordResetChallengeEvent {
	challengeUrl := fmt.Sprintf("%s/auth/reset-password/%s", webUrl, challengeID)
	return PasswordResetChallengeEvent{
		Publisher: publisher,
		Payload: PasswordResetChallengePayload{
			Type:         PasswordResetChallengeName,
			Secret:       secret,
			To:           to,
			WebUrl:       webUrl,
			ChallengeUrl: challengeUrl,
		},
	}
}

func (e *PasswordResetChallengeEvent) Trigger() {
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

func (e *PasswordResetChallengeEvent) callback(params *EventParams) error {
	e.Payload.WebUrl = params.WebUrl
	subject := "Password Reset Request"
	err := params.Notifier.NotifyFromTemplate(e.Payload.To, subject, "password_reset", e.Payload)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
		return err
	}
	return nil
}

const PasswordResetSuccessName = "PasswordResetSuccess"
const PasswordResetSuccessPayloadName = "PasswordResetSuccessPayload"

type PasswordResetSuccessPayload struct {
	Type      string
	Email     string
	WebUrl    string
	ResetDate string
}

type PasswordResetSuccessEvent struct {
	Publisher messaging.IPublisher
	Payload   PasswordResetSuccessPayload
}

func NewPasswordResetSuccess(
	publisher messaging.IPublisher,
	email string,
	webUrl string,
	resetDate string,
) PasswordResetSuccessEvent {
	return PasswordResetSuccessEvent{
		Publisher: publisher,
		Payload: PasswordResetSuccessPayload{
			Type:      PasswordResetSuccessName,
			Email:     email,
			WebUrl:    webUrl,
			ResetDate: resetDate,
		},
	}
}

func (e *PasswordResetSuccessEvent) Trigger() {
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

func (e *PasswordResetSuccessEvent) callback(params *EventParams) error {
	e.Payload.WebUrl = params.WebUrl
	subject := "Password Reset Successful"
	err := params.Notifier.NotifyFromTemplate(e.Payload.Email, subject, "password_reset_success", e.Payload)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
		return err
	}
	return nil
}
