package sql

import (
	"api/internal/errors"
	"api/internal/helpers"
	"api/internal/models"

	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func CreateUserWithRole(
	db *gorm.DB,
	enforcer *casbin.Enforcer,
	user *models.User,
	roleFunc func(*casbin.Enforcer, models.User) error,
) error {
	tx := db.Begin()

	err := CreateUserWithRoleBase(tx, enforcer, user, roleFunc)

	if err != nil {
		tx.Rollback()
		return errors.NewAPIError(500, "CREATE_USER_FAILED")
	}

	tx.Commit()

	return nil
}

func CreateUserWithRoleBase(
	tx *gorm.DB,
	enforcer *casbin.Enforcer,
	user *models.User,
	roleFunc func(*casbin.Enforcer, models.User) error,
) error {
	res := tx.Create(&user)
	if res.Error != nil {
		zap.L().Error("Error creating user", zap.Error(res.Error))
		return errors.NewAPIError(500, "CREATE_USER_FAILED")
	}

	err := roleFunc(enforcer, *user)
	if err != nil {
		zap.L().Error("can not add user to role", zap.Error(err))
		return errors.NewAPIError(500, "CREATE_USER_FAILED")
	}

	err = helpers.AllowUserToSelfModify(enforcer, *user)
	if err != nil {
		zap.L().Error("can not allow user self modify", zap.Error(err))
		return errors.NewAPIError(500, "CREATE_USER_FAILED")
	}

	return nil
}
