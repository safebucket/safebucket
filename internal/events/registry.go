package events

import "reflect"

var eventRegistry = map[string]reflect.Type{
	BucketSharedWithName:                reflect.TypeOf(BucketSharedWith{}),
	BucketSharedWithPayloadName:         reflect.TypeOf(BucketSharedWithPayload{}),
	ChallengeUserInviteName:             reflect.TypeOf(ChallengeUserInvite{}),
	ChallengeUserInvitePayloadName:      reflect.TypeOf(ChallengeUserInvitePayload{}),
	PasswordResetChallengeName:          reflect.TypeOf(PasswordResetChallengeEvent{}),
	PasswordResetChallengePayloadName:   reflect.TypeOf(PasswordResetChallengePayload{}),
	PasswordResetSuccessName:            reflect.TypeOf(PasswordResetSuccessEvent{}),
	PasswordResetSuccessPayloadName:     reflect.TypeOf(PasswordResetSuccessPayload{}),
	UserWelcomeName:                     reflect.TypeOf(UserWelcomeEvent{}),
	UserWelcomePayloadName:              reflect.TypeOf(UserWelcomePayload{}),
	UserInvitationName:                  reflect.TypeOf(UserInvitation{}),
	UserInvitationPayloadName:           reflect.TypeOf(UserInvitationPayload{}),
	BucketPurgeName:                     reflect.TypeOf(BucketPurge{}),
	BucketPurgePayloadName:              reflect.TypeOf(BucketPurgePayload{}),
	TrashExpirationName:                 reflect.TypeOf(TrashExpiration{}),
	TrashExpirationPayloadName:          reflect.TypeOf(TrashExpirationPayload{}),
	FolderRestoreName:                   reflect.TypeOf(FolderRestore{}),
	FolderRestorePayloadName:            reflect.TypeOf(FolderRestorePayload{}),
	FolderTrashName:                     reflect.TypeOf(FolderTrash{}),
	FolderTrashPayloadName:              reflect.TypeOf(FolderTrashPayload{}),
	FolderPurgeName:                     reflect.TypeOf(FolderPurge{}),
	FolderPurgePayloadName:              reflect.TypeOf(FolderPurgePayload{}),
	FileActivityNotificationName:        reflect.TypeOf(FileActivityNotification{}),
	FileActivityNotificationPayloadName: reflect.TypeOf(FileActivityNotificationPayload{}),
}
