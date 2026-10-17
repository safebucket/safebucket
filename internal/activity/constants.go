package activity

var ValidActions []string

func defineAction(name string) string {
	ValidActions = append(ValidActions, name)
	return name
}

var (
	BucketCreated                = defineAction("BUCKET_CREATED")
	BucketUpdated                = defineAction("BUCKET_UPDATED")
	BucketDeleted                = defineAction("BUCKET_DELETED")
	FileUploaded                 = defineAction("FILE_UPLOADED")
	FileDownloaded               = defineAction("FILE_DOWNLOADED")
	FileDeleted                  = defineAction("FILE_DELETED")
	FileExpired                  = defineAction("FILE_EXPIRED")
	FileTrashed                  = defineAction("FILE_TRASHED")
	FileRestored                 = defineAction("FILE_RESTORED")
	FolderCreated                = defineAction("FOLDER_CREATED")
	FolderUpdated                = defineAction("FOLDER_UPDATED")
	FolderTrashed                = defineAction("FOLDER_TRASHED")
	FolderRestored               = defineAction("FOLDER_RESTORED")
	FolderDeleted                = defineAction("FOLDER_DELETED")
	BucketMemberCreated          = defineAction("BUCKET_MEMBER_CREATED")
	BucketMemberUpdated          = defineAction("BUCKET_MEMBER_UPDATED")
	BucketMemberDeleted          = defineAction("BUCKET_MEMBER_DELETED")
	UserCreated                  = defineAction("USER_CREATED")
	UserLoggedIn                 = defineAction("USER_LOGGED_IN")
	UserDeleted                  = defineAction("USER_DELETED")
	PasswordResetCodeVerified    = defineAction("PASSWORD_RESET_CODE_VERIFIED")
	PasswordResetCompleted       = defineAction("PASSWORD_RESET_COMPLETED")
	InviteAccepted               = defineAction("INVITE_ACCEPTED")
	InviteChallengeAttemptFailed = defineAction("INVITE_CHALLENGE_ATTEMPT_FAILED")
	InviteChallengeLocked        = defineAction("INVITE_CHALLENGE_LOCKED")
	MFADeviceEnrolled            = defineAction("MFA_DEVICE_ENROLLED")
	MFADeviceVerified            = defineAction("MFA_DEVICE_VERIFIED")
	MFADeviceUpdated             = defineAction("MFA_DEVICE_UPDATED")
	MFADeviceRemoved             = defineAction("MFA_DEVICE_REMOVED")
	SessionRevoked               = defineAction("SESSION_REVOKED")
	OtherSessionsRevoked         = defineAction("OTHER_SESSIONS_REVOKED")
	ShareCreated                 = defineAction("SHARE_CREATED")
	ShareDeleted                 = defineAction("SHARE_DELETED")
	ShareExpired                 = defineAction("SHARE_EXPIRED")
	ShareMaxViewsReached         = defineAction("SHARE_MAX_VIEWS_REACHED")
	ShareFileDownloaded          = defineAction("SHARE_FILE_DOWNLOADED")
	ShareFileUploaded            = defineAction("SHARE_FILE_UPLOADED")
)
