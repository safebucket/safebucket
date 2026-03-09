package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Group string

const (
	GroupOwner       Group = "owner"
	GroupContributor Group = "contributor"
	GroupViewer      Group = "viewer"
)

type Membership struct {
	ID                    uuid.UUID      `gorm:"default:(-)"                                     json:"id"`
	UserID                uuid.UUID      `gorm:"not null;uniqueIndex:idx_user_bucket"            json:"user_id"`
	User                  User           `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"   json:"user,omitempty"`
	BucketID              uuid.UUID      `gorm:"not null;uniqueIndex:idx_user_bucket"            json:"bucket_id"`
	Bucket                Bucket         `gorm:"foreignKey:BucketID;constraint:OnDelete:CASCADE" json:"bucket,omitempty"`
	Group                 Group          `gorm:"not null"                                        json:"group"                  validate:"required,oneof=owner contributor viewer"`
	UploadNotifications   bool           `gorm:"not null;default:true"                           json:"upload_notifications"`
	DownloadNotifications bool           `gorm:"not null;default:false"                          json:"download_notifications"`
	CreatedAt             time.Time      `                                                       json:"created_at"`
	UpdatedAt             time.Time      `                                                       json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index"                                           json:"-"`
}

type MembershipCreateBody struct {
	UserID   uuid.UUID `json:"user_id"   validate:"required"`
	BucketID uuid.UUID `json:"bucket_id" validate:"required"`
	Group    Group     `json:"group"     validate:"required,oneof=owner contributor viewer"`
}

type MembershipUpdateBody struct {
	Group Group `json:"group" validate:"required,oneof=owner contributor viewer"`
}

type MembershipNotificationBody struct {
	UploadNotifications   *bool `json:"upload_notifications"   validate:"required,boolean"`
	DownloadNotifications *bool `json:"download_notifications" validate:"required,boolean"`
}
