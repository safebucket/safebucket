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
	}
}

func AddUserToOwners(e *casbin.Enforcer, bucket models.Bucket, userId string) error {
	_, err := e.AddGroupingPolicy(userId, GetBucketOwnerGroup(bucket), c.DefaultDomain)
	if err != nil {
		zap.L().Error("Failed to add user to owners", zap.Error(err))
		return err
	}
	return nil
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
