package groups

import (
	c "api/internal/configuration"
	"api/internal/models"
	"api/internal/rbac"
	"fmt"

	"github.com/casbin/casbin/v2"
)

func GetBucketViewerGroup(bucket models.Bucket) string {
	return fmt.Sprintf("%s::%s", rbac.GroupViewer, bucket.ID.String())
}

func GetDefaultViewerBucketPolicies(bucket models.Bucket) [][]string {
	groupName := GetBucketViewerGroup(bucket)
	return [][]string{
		{c.DefaultDomain, groupName, rbac.ResourceBucket.String(), bucket.ID.String(), rbac.ActionRead.String()},
	}
}

func InsertGroupBucketViewer(e *casbin.Enforcer, bucket models.Bucket) error {
	_, err := e.AddPolicies(GetDefaultViewerBucketPolicies(bucket))
	if err != nil {
		return err
	}
	return nil
}
