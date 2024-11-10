package models

import (
	"github.com/google/uuid"
)

//type Policies struct {
//	ID    uint      `gorm:"primaryKey;autoIncrement"`
//	Ptype string    `gorm:"size:512;uniqueIndex:unique_index"`
//	V0    uuid.UUID `gorm:"type:uuid;uniqueIndex:unique_index"`
//	V1    uint      `gorm:"uniqueIndex:unique_index"`
//	V2    uuid.UUID `gorm:"type:uuid;uniqueIndex:unique_index"`
//	V3    string    `gorm:"size:512;uniqueIndex:unique_index"`
//	V4    string    `gorm:"size:512;uniqueIndex:unique_index"`
//	V5    string    `gorm:"size:512;uniqueIndex:unique_index"`
//}

type Policies struct {
	ID    uint      `gorm:"primaryKey;autoIncrement"`
	Ptype string    `gorm:"size:512;uniqueIndex:unique_index"`
	V0    uuid.UUID `gorm:"type:uuid;uniqueIndex:unique_index"`
	V1    uint16    `gorm:"type:uuid;uniqueIndex:unique_index"`
	V2    uuid.UUID `gorm:"type:uuid;uniqueIndex:unique_index"`
	V3    string    `gorm:"size:512;uniqueIndex:unique_index"`
	V4    string    `gorm:"size:512;uniqueIndex:unique_index"`
	V5    string    `gorm:"size:512;uniqueIndex:unique_index"`
}
