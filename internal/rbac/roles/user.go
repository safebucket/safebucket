package roles

import (
	c "api/internal/configuration"
	"api/internal/models"
	"api/internal/rbac"
	"github.com/casbin/casbin/v2"
)

func getDefaultUserPolicies() [][]string {
	return [][]string{
		{c.DefaultDomain, rbac.RoleUser.String(), rbac.ResourceBucket.String(), c.NilUUID, rbac.ActionList.String()},
		{c.DefaultDomain, rbac.RoleUser.String(), rbac.ResourceBucket.String(), c.NilUUID, rbac.ActionCreate.String()},
		{c.DefaultDomain, rbac.RoleUser.String(), rbac.ResourceUser.String(), c.NilUUID, rbac.ActionList.String()},
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
	_, err := e.AddGroupingPolicy(user.ID.String(), rbac.RoleUser.String(), c.DefaultDomain)
	if err != nil {
		return err
	}
	return nil
}

func AllowUserToSelfModify(e *casbin.Enforcer, user models.User) error {
	{
		_, err := e.AddPolicies(getUserPoliciesToSelfModify(user))
		if err != nil {
			return err
		}
		return nil
	}
}

func getUserPoliciesToSelfModify(user models.User) [][]string {
	return [][]string{
		{c.DefaultDomain, user.ID.String(), rbac.ResourceUser.String(), user.ID.String(), rbac.ActionRead.String()},
		{c.DefaultDomain, user.ID.String(), rbac.ResourceUser.String(), user.ID.String(), rbac.ActionUpdate.String()},
	}
}
