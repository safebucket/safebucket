package groups

import (
	c "api/internal/configuration"
	"api/internal/models"
	"api/internal/rbac"
	"fmt"
	"github.com/casbin/casbin/v2"
)

func GetBucketOwnerGroup(bucket models.Bucket) string {
	return fmt.Sprintf("%s::%s", rbac.GroupOwner, bucket.ID.String())
}

func GetDefaultOwnerBucketPolicies(bucket models.Bucket) [][]string {
	groupName := GetBucketOwnerGroup(bucket)
	return [][]string{
		{c.DefaultDomain, groupName, rbac.ResourceBucket, bucket.ID.String(), rbac.ActionDelete},
		{c.DefaultDomain, groupName, rbac.ResourceBucket, bucket.ID.String(), rbac.ActionUpdate},
	}
}

func AddUserToOwners(e *casbin.Enforcer, bucket models.Bucket, claims *models.UserClaims) error {
	_, err := e.AddGroupingPolicy(claims.UserID, GetBucketOwnerGroup(bucket), c.DefaultDomain)
	if err != nil {
		return err
	}
}

func InsertGroupOwner(e *casbin.Enforcer, bucket models.Bucket) error {
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
