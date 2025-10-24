package groups

import (
	c "api/internal/configuration"
	"api/internal/models"
	"api/internal/rbac"
	"fmt"

	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
)

func GetBucketContributorGroup(bucket models.Bucket) string {
	return fmt.Sprintf("%s::%s", rbac.GroupContributor, bucket.ID.String())
}

func GetDefaultContributorBucketPolicies(bucket models.Bucket) [][]string {
	groupName := GetBucketContributorGroup(bucket)
	return [][]string{
		{c.DefaultDomain, groupName, rbac.ResourceBucket.String(), bucket.ID.String(), rbac.ActionUpload.String()},
		{c.DefaultDomain, groupName, rbac.ResourceBucket.String(), bucket.ID.String(), rbac.ActionErase.String()},
		{c.DefaultDomain, groupName, rbac.ResourceBucket.String(), bucket.ID.String(), rbac.ActionRestore.String()},
	}
}

func InsertGroupBucketContributor(e *casbin.Enforcer, bucket models.Bucket) error {
	_, err := e.AddPolicies(GetDefaultContributorBucketPolicies(bucket))
	if err != nil {
		return err
	}
	_, err = e.AddGroupingPolicy(GetBucketContributorGroup(bucket), GetBucketViewerGroup(bucket), c.DefaultDomain)
	if err != nil {
		return err
	}
	return nil
}

func RemoveGroupBucketContributor(e *casbin.Enforcer, bucket models.Bucket) error {
	_, err := e.RemovePolicies(GetDefaultContributorBucketPolicies(bucket))
	if err != nil {
		return err
	}
	_, err = e.RemoveGroupingPolicy(GetBucketContributorGroup(bucket), GetBucketViewerGroup(bucket), c.DefaultDomain)
	if err != nil {
		return err
	}
	return nil
}

func AddUserToContributors(e *casbin.Enforcer, bucket models.Bucket, userId string) error {
	_, err := e.AddGroupingPolicy(userId, GetBucketContributorGroup(bucket), c.DefaultDomain)
	if err != nil {
		zap.L().Error("Failed to add user to contributors", zap.Error(err))
		return err
	}
	return nil
}

func RemoveUserFromContributors(e *casbin.Enforcer, bucket models.Bucket, userId string) error {
	_, err := e.RemoveGroupingPolicy(userId, GetBucketContributorGroup(bucket), c.DefaultDomain)
	if err != nil {
		zap.L().Error("Failed to remove user from contributors", zap.Error(err))
		return err
	}
	return nil
}

func RemoveUsersFromContributors(e *casbin.Enforcer, bucket models.Bucket) error {
	_, err := e.RemoveGroupingPolicy("", GetBucketContributorGroup(bucket), c.DefaultDomain)
	if err != nil {
		zap.L().Error("Failed to remove user from contributors", zap.Error(err))
		return err
	}
	return nil
}
