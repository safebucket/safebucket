package groups

import (
	c "api/internal/configuration"
	"api/internal/models"
	"api/internal/rbac"
	roles "api/internal/rbac/groups"
	"fmt"
	"github.com/casbin/casbin/v2"
)

func GetBucketContributorGroup(bucket models.Bucket) string {
	return fmt.Sprintf("%s::%s", rbac.GroupContributor, bucket.ID.String())
}

func GetDefaultContributorBucketPolicies(bucket models.Bucket) [][]string {
	groupName := GetBucketContributorGroup(bucket)
	return [][]string{
		{c.DefaultDomain, groupName, rbac.ResourceBucket, bucket.ID.String(), rbac.ActionCreate},
	}
}

func InsertGroupBucketContributor(e *casbin.Enforcer, bucket models.Bucket) error {
	_, err := e.AddPolicies(GetDefaultContributorBucketPolicies(bucket))
	if err != nil {
		return err
	}
	_, err = e.AddGroupingPolicy(GetBucketContributorGroup(bucket), roles.GetBucketViewverGroup(bucket), c.DefaultDomain)
	if err != nil {
		return err
	}
	return nil
}
