package groups

import (
	c "api/internal/configuration"
	"api/internal/models"
	"api/internal/rbac"
	"fmt"

	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
)

func GetBucketOwnerGroup(bucket models.Bucket) string {
	return fmt.Sprintf("%s::%s", rbac.GroupOwner, bucket.ID.String())
}

func GetDefaultOwnerBucketPolicies(bucket models.Bucket) [][]string {
	groupName := GetBucketOwnerGroup(bucket)
	return [][]string{
		{c.DefaultDomain, groupName, rbac.ResourceBucket.String(), bucket.ID.String(), rbac.ActionDelete.String()},
		{c.DefaultDomain, groupName, rbac.ResourceBucket.String(), bucket.ID.String(), rbac.ActionUpdate.String()},
		{c.DefaultDomain, groupName, rbac.ResourceBucket.String(), bucket.ID.String(), rbac.ActionGrant.String()},
		{c.DefaultDomain, groupName, rbac.ResourceBucket.String(), bucket.ID.String(), rbac.ActionPurge.String()},
	}
}

func InsertGroupBucketOwner(e *casbin.Enforcer, bucket models.Bucket) error {
	_, err := e.AddPolicies(GetDefaultOwnerBucketPolicies(bucket))
	if err != nil {
		return err
	}
	_, err = e.AddGroupingPolicy(GetBucketOwnerGroup(bucket), GetBucketContributorGroup(bucket), c.DefaultDomain)
	if err != nil {
		return err
	}
	return nil
}

func RemoveGroupBucketOwner(e *casbin.Enforcer, bucket models.Bucket) error {
	_, err := e.RemovePolicies(GetDefaultOwnerBucketPolicies(bucket))
	if err != nil {
		return err
	}
	_, err = e.RemoveGroupingPolicy(GetBucketOwnerGroup(bucket), GetBucketContributorGroup(bucket), c.DefaultDomain)
	if err != nil {
		return err
	}
	return nil
}

func AddUserToOwners(e *casbin.Enforcer, bucket models.Bucket, userId string) error {
	_, err := e.AddGroupingPolicy(userId, GetBucketOwnerGroup(bucket), c.DefaultDomain)
	if err != nil {
		zap.L().Error("Failed to add user to owners", zap.Error(err))
		return err
	}
	return nil
}

func RemoveUserFromOwners(e *casbin.Enforcer, bucket models.Bucket, userId string) error {
	_, err := e.RemoveGroupingPolicy(userId, GetBucketOwnerGroup(bucket), c.DefaultDomain)
	if err != nil {
		zap.L().Error("Failed to remove user from owners", zap.Error(err))
		return err
	}
	return nil
}

func RemoveUsersFromOwners(e *casbin.Enforcer, bucket models.Bucket) error {
	_, err := e.RemoveGroupingPolicy("", GetBucketOwnerGroup(bucket), c.DefaultDomain)
	if err != nil {
		zap.L().Error("Failed to remove user from owners", zap.Error(err))
		return err
	}
	return nil
}
