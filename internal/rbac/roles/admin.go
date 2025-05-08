package roles

import (
	c "api/internal/configuration"
	"api/internal/models"
	"api/internal/rbac"
	"github.com/casbin/casbin/v2"
)

func getDefaultAdminPolicies() [][]string {
	return [][]string{
		{c.DefaultDomain, rbac.RoleUser.String(), rbac.ResourceAll.String(), c.NilUUID, rbac.ActionAll.String()},
	}
}

func InsertRoleAdmin(e *casbin.Enforcer) error {
	_, err := e.AddPolicies(getDefaultAdminPolicies())
	if err != nil {
		return err
	}
	return nil
}

func AddUserToRoleAdmin(e *casbin.Enforcer, user models.User) error {
	_, err := e.AddGroupingPolicy(
		user.ID.String(), rbac.RoleAdmin.String(), c.DefaultDomain)
	if err != nil {
		return err
	}

	_, err = e.AddGroupingPolicy(
		user.ID.String(), rbac.RoleUser.String(), c.DefaultDomain)
	if err != nil {
		return err
	}

	return nil
}
