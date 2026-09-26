package events

import (
	"strconv"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/fileversions"
	"github.com/safebucket/safebucket/internal/messaging"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func NotifyFileVersionUploaded(logger *zap.Logger, db *gorm.DB, activityLogger activity.IActivityLogger,
	publisher messaging.IPublisher, completed fileversions.Completion,
) {
	file, version := completed.File, completed.Version
	fields := models.ActivityFields{Action: rbac.ActionCreate.String(), ObjectType: rbac.ResourceFile.String(),
		BucketID: file.BucketID.String(), FileID: file.ID.String(), Version: strconv.Itoa(version.Version)}
	action := activity.FileUploaded
	if version.Version > 1 {
		action = activity.FileVersionCreated
	}
	source := FileActivitySourceUser
	var userID uuid.UUID
	if version.UploadedBy != nil {
		userID = *version.UploadedBy
		fields.UserID = userID.String()
	}
	if version.ShareID != nil {
		fields.ShareID = version.ShareID.String()
		if version.Version == 1 {
			action = activity.ShareFileUploaded
		}
		source = FileActivitySourceShare
		var share models.Share
		if result := db.Where("id = ?", version.ShareID).Find(&share); result.Error == nil && result.RowsAffected > 0 {
			userID = share.CreatedBy
		}
	}
	if err := activityLogger.Send(
		models.Activity{Message: action, Object: file.ToActivity(), Filter: activity.NewLogFilter(fields)},
	); err != nil {
		logger.Warn("Failed to log version upload", zap.Error(err))
	}
	if userID == uuid.Nil {
		return
	}
	var user models.User
	var bucket models.Bucket
	if result := db.Where("id = ?", userID).Find(&user); result.Error != nil || result.RowsAffected == 0 {
		return
	}
	if result := db.Where("id = ?", file.BucketID).Find(&bucket); result.Error != nil || result.RowsAffected == 0 {
		return
	}
	event := NewFileActivityNotification(
		publisher,
		FileActivityUpload,
		source,
		file.BucketID,
		bucket.Name,
		file.Name,
		userID,
		user.Email,
	)
	event.Trigger()
}
