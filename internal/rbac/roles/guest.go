package roles

import (
	c "api/internal/configuration"
	"api/internal/models"
	"api/internal/rbac"
	"github.com/casbin/casbin/v2"
)

func getDefaultGuestPolicies() [][]string {
	return [][]string{
		{c.DefaultDomain, rbac.RoleGuest, rbac.ResourceBucket, c.NilUUID, rbac.ActionList},
	}
}

func InsertRoleGuest(e *casbin.Enforcer) error {
	_, err := e.AddPolicies(getDefaultGuestPolicies())
	if err != nil {
		return err
	}
	return nil
}

func AddUserToRoleGuest(e *casbin.Enforcer, user models.User) error {
	_, err := e.AddGroupingPolicy(user.ID, rbac.RoleGuest, c.DefaultDomain)
	if err != nil {
		return err
	}
	return nil
}
