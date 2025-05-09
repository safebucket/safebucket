package groups

import (
	c "api/internal/configuration"
	"api/internal/models"
	"api/internal/rbac"
	"fmt"
	"go.uber.org/zap"

	"github.com/casbin/casbin/v2"
)

func GetBucketViewerGroup(bucket models.Bucket) string {
	return fmt.Sprintf("%s::%s", rbac.GroupViewer, bucket.ID.String())
}

func GetDefaultViewerBucketPolicies(bucket models.Bucket) [][]string {
	groupName := GetBucketViewerGroup(bucket)
	return [][]string{
		{c.DefaultDomain, groupName, rbac.ResourceBucket.String(), bucket.ID.String(), rbac.ActionRead.String()},
		{c.DefaultDomain, groupName, rbac.ResourceBucket.String(), bucket.ID.String(), rbac.ActionDownload.String()},
	}
}

func AddUserToViewers(e *casbin.Enforcer, bucket models.Bucket, userId string) error {
	_, err := e.AddGroupingPolicy(userId, GetBucketViewerGroup(bucket), c.DefaultDomain)
	if err != nil {
		zap.L().Error("Failed to add user to viewers", zap.Error(err))
		return err
	}
	return nil
}

func InsertGroupBucketViewer(e *casbin.Enforcer, bucket models.Bucket) error {
	_, err := e.AddPolicies(GetDefaultViewerBucketPolicies(bucket))
	if err != nil {
		return err
	}
	return nil
}
