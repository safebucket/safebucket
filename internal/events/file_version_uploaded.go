package events

import (
	"strconv"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/fileversions"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"
	"github.com/safebucket/safebucket/internal/rbac"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func NotifyFileVersionUploaded(
	logger *zap.Logger, db *gorm.DB, completed fileversions.Completion, shareID *uuid.UUID,
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
	if shareID != nil {
		fields.ShareID = shareID.String()
		if version.Version == 1 {
			action = activity.ShareFileUploaded
		}
		source = FileActivitySourceShare
		var share models.Share
		if result := db.Where("id = ?", shareID).Find(&share); result.Error == nil && result.RowsAffected > 0 {
			userID = share.CreatedBy
		}
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

	uploaded := models.Activity{Message: action, Object: file.ToActivity(), Filter: activity.NewLogFilter(fields)}

	notification := NewFileActivityNotification(
		FileActivityUpload,
		source,
		file.BucketID,
		bucket.Name,
		file.Name,
		userID,
		user.Email,
	)

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := outbox.AddActivities(tx, uploaded); err != nil {
			return err
		}
		return outbox.AddEvent(tx, FileActivityNotificationName, notification.Payload)
	}); err != nil {
		logger.Error("Failed to record version upload", zap.Error(err))
	}
}
