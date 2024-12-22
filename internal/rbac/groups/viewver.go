package roles

import (
	c "api/internal/configuration"
	"api/internal/models"
	"api/internal/rbac"
	"fmt"

	"github.com/casbin/casbin/v2"
)

func GetBucketViewverGroup(bucket models.Bucket) string {
	return fmt.Sprintf("%s::%s", rbac.GroupViewer, bucket.ID.String())
}

func GetDefaultViewverBucketPolicies(bucket models.Bucket) [][]string {
	groupName := GetBucketViewverGroup(bucket)
	return [][]string{
		{c.DefaultDomain, groupName, rbac.ResourceBucket, bucket.ID.String(), rbac.ActionRead},
	}
}

func InsertGroupBucketViewver(e *casbin.Enforcer, bucket models.Bucket) error {
	_, err := e.AddPolicies(GetDefaultViewverBucketPolicies(bucket))
	if err != nil {
		return err
	}
	return nil
}
