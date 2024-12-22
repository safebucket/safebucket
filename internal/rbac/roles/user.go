package roles

import (
	c "api/internal/configuration"
	"api/internal/models"
	"api/internal/rbac"

	"github.com/casbin/casbin/v2"
)

func getDefaultUserPolicies() [][]string {
	return [][]string{
		{c.DefaultDomain, rbac.RoleUser, rbac.ResourceBucket, c.NilUUID, rbac.ActionList},
		{c.DefaultDomain, rbac.RoleUser, rbac.ResourceBucket, c.NilUUID, rbac.ActionCreate},
	}
}

func InsertRoleUser(e *casbin.Enforcer) error {
	_, err := e.AddPolicies(getDefaultUserPolicies())
	if err != nil {
		return err
	}
	return nil
}

func AddUserToRoleUser(e *casbin.Enforcer, user models.User) error {
	_, err := e.AddGroupingPolicy(user.ID, rbac.RoleUser, c.DefaultDomain)
	if err != nil {
		return err
	}
	return nil
}
