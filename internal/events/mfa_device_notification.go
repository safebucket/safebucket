package events

import (
	"go.uber.org/zap"
)

const (
	MFADeviceNotificationName        = "MFADeviceNotification"
	MFADeviceNotificationPayloadName = "MFADeviceNotificationPayload"
)

type MFADeviceAction string

const (
	MFADeviceEnrolled MFADeviceAction = "enrolled"
	MFADeviceRemoved  MFADeviceAction = "removed"
)

type MFADeviceNotificationPayload struct {
	Type       string
	Action     MFADeviceAction
	To         string
	DeviceName string
	WebURL     string
}

func (p MFADeviceNotificationPayload) EventName() string { return p.Type }

type MFADeviceNotification struct {
	Payload MFADeviceNotificationPayload
}

func NewMFADeviceNotification(action MFADeviceAction, to string, deviceName string) MFADeviceNotification {
	return MFADeviceNotification{
		Payload: MFADeviceNotificationPayload{
			Type:       MFADeviceNotificationName,
			Action:     action,
			To:         to,
			DeviceName: deviceName,
		},
	}
}

func (e *MFADeviceNotification) callback(params *EventParams) error {
	subject, template := "New MFA Device Enrolled", "mfa_device_enrolled"
	if e.Payload.Action == MFADeviceRemoved {
		subject, template = "MFA Device Removed", "mfa_device_removed"
	}

	e.Payload.WebURL = params.WebURL
	err := params.Notifier.NotifyFromTemplate(e.Payload.To, subject, template, e.Payload)
	if err != nil {
		zap.L().Error("failed to notify", zap.Any("event", e), zap.Error(err))
		return err
	}
	return nil
}
