package models

import "github.com/google/uuid"

type BucketMemberBody struct {
	Email string `json:"email" validate:"required,email,max=254"`
	Group Group  `json:"group" validate:"required,oneof=owner contributor viewer"`
}

type UpdateMembersBody struct {
	Members []BucketMemberBody `json:"members" validate:"required,min=1,max=1000,dive"`
}

type BucketMember struct {
	UserID    uuid.UUID `json:"user_id,omitempty"`
	Email     string    `json:"email"                validate:"required"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	Group     Group     `json:"group"                validate:"required,oneof=owner contributor viewer"`
	Status    string    `json:"status"               validate:"required,oneof=active invited"`
}

type BucketMemberToUpdate struct {
	BucketMember

	NewGroup Group `validate:"required,oneof=owner contributor viewer"`
}

type MembershipChanges struct {
	ToAdd    []BucketMemberBody
	ToUpdate []BucketMemberToUpdate
	ToDelete []BucketMember
}
