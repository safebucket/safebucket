package helpers

import (
	c "api/internal/configuration"
	"api/internal/models"
	"api/internal/rbac"

	"github.com/casbin/casbin/v2"
)

func getUserPoliciesToSelfModify(user models.User) [][]string {
	return [][]string{
		{c.DefaultDomain, user.ID.String(), rbac.ResourceUser.String(), user.ID.String(), rbac.ActionRead.String()},
		{c.DefaultDomain, user.ID.String(), rbac.ResourceUser.String(), user.ID.String(), rbac.ActionUpdate.String()},
	}
}

func AllowUserToSelfModify(e *casbin.Enforcer, user models.User) error {
	_, err := e.AddPolicies(getUserPoliciesToSelfModify(user))
	if err != nil {
		return err
	}
	return nil
}
