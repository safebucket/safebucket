package events

import (
	"fmt"

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
	Payload ChallengeUserInvitePayload
}

func NewChallengeUserInvite(
	secret string,
	to string,
	from string,
	inviteID string,
	challengeID string,
	webURL string,
) ChallengeUserInvite {
	challengeURL := fmt.Sprintf("%s/invites/%s/challenges/%s", webURL, inviteID, challengeID)
	return ChallengeUserInvite{
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
	Payload PasswordResetChallengePayload
}

func NewPasswordResetChallenge(
	secret string,
	to string,
	challengeID string,
	webURL string,
) PasswordResetChallengeEvent {
	challengeURL := fmt.Sprintf("%s/auth/reset-password/%s", webURL, challengeID)
	return PasswordResetChallengeEvent{
		Payload: PasswordResetChallengePayload{
			Type:         PasswordResetChallengeName,
			Secret:       secret,
			To:           to,
			WebURL:       webURL,
			ChallengeURL: challengeURL,
		},
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
	Payload PasswordResetSuccessPayload
}

func NewPasswordResetSuccess(
	email string,
	webURL string,
	resetDate string,
) PasswordResetSuccessEvent {
	return PasswordResetSuccessEvent{
		Payload: PasswordResetSuccessPayload{
			Type:      PasswordResetSuccessName,
			Email:     email,
			WebURL:    webURL,
			ResetDate: resetDate,
		},
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
	Payload UserWelcomePayload
}

func NewUserWelcome(
	email string,
	webURL string,
) UserWelcomeEvent {
	return UserWelcomeEvent{
		Payload: UserWelcomePayload{
			Type:   UserWelcomeName,
			Email:  email,
			WebURL: webURL,
		},
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
