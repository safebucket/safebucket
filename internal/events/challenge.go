package events

import (
	"encoding/json"
	"fmt"

	"api/internal/messaging"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

const (
	ChallengeUserInviteName        = "ChallengeUserInvite"
	ChallengeUserInvitePayloadName = "ChallengeUserInvitePayload"
)

type ChallengeUserInvitePayload struct {
	Type         string
	Secret       string
	To           string
	From         string
	WebURL       string
	ChallengeURL string
}

type ChallengeUserInvite struct {
	Publisher messaging.IPublisher
	Payload   ChallengeUserInvitePayload
}

func NewChallengeUserInvite(
	publisher messaging.IPublisher,
	secret string,
	to string,
	from string,
	inviteID string,
	challengeID string,
	webURL string,
) ChallengeUserInvite {
	challengeURL := fmt.Sprintf("%s/invites/%s/challenges/%s", webURL, inviteID, challengeID)
	return ChallengeUserInvite{
		Publisher: publisher,
		Payload: ChallengeUserInvitePayload{
			Type:         ChallengeUserInviteName,
			Secret:       secret,
			To:           to,
			From:         from,
			WebURL:       webURL,
			ChallengeURL: challengeURL,
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
	e.Payload.WebURL = params.WebURL
	subject := fmt.Sprintf("%s has invited you", e.Payload.From)
	err := params.Notifier.NotifyFromTemplate(e.Payload.To, subject, "user_invited", e.Payload)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
		return err
	}
	return nil
}

const (
	PasswordResetChallengeName        = "PasswordResetChallenge"
	PasswordResetChallengePayloadName = "PasswordResetChallengePayload"
)

type PasswordResetChallengePayload struct {
	Type         string
	Secret       string
	To           string
	WebURL       string
	ChallengeURL string
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
	webURL string,
) PasswordResetChallengeEvent {
	challengeURL := fmt.Sprintf("%s/auth/reset-password/%s", webURL, challengeID)
	return PasswordResetChallengeEvent{
		Publisher: publisher,
		Payload: PasswordResetChallengePayload{
			Type:         PasswordResetChallengeName,
			Secret:       secret,
			To:           to,
			WebURL:       webURL,
			ChallengeURL: challengeURL,
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
	e.Payload.WebURL = params.WebURL
	subject := "Password Reset Request"
	err := params.Notifier.NotifyFromTemplate(e.Payload.To, subject, "password_reset", e.Payload)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
		return err
	}
	return nil
}

const (
	PasswordResetSuccessName        = "PasswordResetSuccess"
	PasswordResetSuccessPayloadName = "PasswordResetSuccessPayload"
)

type PasswordResetSuccessPayload struct {
	Type      string
	Email     string
	WebURL    string
	ResetDate string
}

type PasswordResetSuccessEvent struct {
	Publisher messaging.IPublisher
	Payload   PasswordResetSuccessPayload
}

func NewPasswordResetSuccess(
	publisher messaging.IPublisher,
	email string,
	webURL string,
	resetDate string,
) PasswordResetSuccessEvent {
	return PasswordResetSuccessEvent{
		Publisher: publisher,
		Payload: PasswordResetSuccessPayload{
			Type:      PasswordResetSuccessName,
			Email:     email,
			WebURL:    webURL,
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
	e.Payload.WebURL = params.WebURL
	subject := "Password Reset Successful"
	err := params.Notifier.NotifyFromTemplate(
		e.Payload.Email,
		subject,
		"password_reset_success",
		e.Payload,
	)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
		return err
	}
	return nil
}

const (
	MFAResetChallengeName        = "MFAResetChallenge"
	MFAResetChallengePayloadName = "MFAResetChallengePayload"
)

type MFAResetChallengePayload struct {
	Type         string
	Secret       string
	To           string
	WebURL       string
	ChallengeURL string
}

type MFAResetChallengeEvent struct {
	Publisher messaging.IPublisher
	Payload   MFAResetChallengePayload
}

func NewMFAResetChallenge(
	publisher messaging.IPublisher,
	secret string,
	to string,
	challengeID string,
	webURL string,
) MFAResetChallengeEvent {
	challengeURL := fmt.Sprintf("%s/auth/mfa-reset/%s", webURL, challengeID)
	return MFAResetChallengeEvent{
		Publisher: publisher,
		Payload: MFAResetChallengePayload{
			Type:         MFAResetChallengeName,
			Secret:       secret,
			To:           to,
			WebURL:       webURL,
			ChallengeURL: challengeURL,
		},
	}
}

func (e *MFAResetChallengeEvent) Trigger() {
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

func (e *MFAResetChallengeEvent) callback(params *EventParams) error {
	e.Payload.WebURL = params.WebURL
	subject := "MFA Device Reset Request"
	err := params.Notifier.NotifyFromTemplate(e.Payload.To, subject, "mfa_reset", e.Payload)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
		return err
	}
	return nil
}

const (
	UserWelcomeName        = "UserWelcome"
	UserWelcomePayloadName = "UserWelcomePayload"
)

type UserWelcomePayload struct {
	Type   string
	Email  string
	WebURL string
}

type UserWelcomeEvent struct {
	Publisher messaging.IPublisher
	Payload   UserWelcomePayload
}

func NewUserWelcome(
	publisher messaging.IPublisher,
	email string,
	webURL string,
) UserWelcomeEvent {
	return UserWelcomeEvent{
		Publisher: publisher,
		Payload: UserWelcomePayload{
			Type:   UserWelcomeName,
			Email:  email,
			WebURL: webURL,
		},
	}
}

func (e *UserWelcomeEvent) Trigger() {
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

func (e *UserWelcomeEvent) callback(params *EventParams) error {
	e.Payload.WebURL = params.WebURL
	subject := "Welcome to Safebucket!"
	err := params.Notifier.NotifyFromTemplate(e.Payload.Email, subject, "user_welcome", e.Payload)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
		return err
	}
	return nil
}
