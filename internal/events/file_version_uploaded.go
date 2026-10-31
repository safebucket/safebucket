package events

import (
	"strconv"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/fileversions"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"
	"github.com/safebucket/safebucket/internal/rbac"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func RecordFileVersionUploaded(shareID *uuid.UUID) func(tx *gorm.DB, completed fileversions.Completion) error {
	return func(tx *gorm.DB, completed fileversions.Completion) error {
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
			result := tx.Where("id = ?", shareID).Find(&share)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				userID = share.CreatedBy
			}
		}

		uploaded := models.Activity{Message: action, Object: file.ToActivity(), Filter: activity.NewLogFilter(fields)}
		err := outbox.AddActivities(tx, uploaded)
		if err != nil || userID == uuid.Nil {
			return err
		}

		var user models.User
		result := tx.Where("id = ?", userID).Find(&user)
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		var bucket models.Bucket
		result = tx.Where("id = ?", file.BucketID).Find(&bucket)
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}

		notification := NewFileActivityNotification(
			FileActivityUpload,
			source,
			file.BucketID,
			bucket.Name,
			file.Name,
			userID,
			user.Email,
		)
		return outbox.AddEvents(tx, FileActivityNotificationName, notification.Payload)
	}
}
