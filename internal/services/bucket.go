package services

import (
	"api/internal/configuration"
	"api/internal/handlers"
	h "api/internal/helpers"
	"api/internal/models"
	a "api/internal/rbac"
	roles "api/internal/rbac/groups"
	"errors"
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BucketService struct {
	DB      *gorm.DB
	JWTConf models.JWTConfiguration
	E       *casbin.Enforcer
}

func (s BucketService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(a.Authenticate(s.JWTConf)).
		With(a.Authorize(models.Operation{Object: a.ResourceBucket, ObjectId: configuration.NilUUID, Action: a.ActionCreate}, s.E)).
		With(h.Validate[models.Bucket]).
		Post("/", handlers.CreateHandler(s.CreateBucket))

	r.With(a.Authenticate(s.JWTConf)).
		With(h.Validate[models.Bucket]).
		Get("/", handlers.GetListHandler(s.GetBucketList)) //TODO: Shall we add list permission ?

	r.Route("/{id}", func(r chi.Router) {
		// TODO: Add With(a.Authorize here)

		r.With(a.Authenticate(s.JWTConf)).
			Get("/", handlers.GetOneHandler(s.GetBucket))

		r.With(a.Authenticate(s.JWTConf)).
			With(h.Validate[models.Bucket]).
			Patch("/", handlers.UpdateHandler(s.UpdateBucket))

		r.With(a.Authenticate(s.JWTConf)).
			Delete("/", handlers.DeleteHandler(s.DeleteBucket))
	})
	return r
}

func (s BucketService) CreateBucket(claims *models.UserClaims, body models.Bucket) (models.Bucket, error) {
	s.DB.Create(&body)

	err := roles.InsertGroupBucketViewver(s.E, body)
	if err != nil {
		return models.Bucket{}, err
	}

	err = roles.InsertGroupBucketContributor(s.E, body)
	if err != nil {
		return models.Bucket{}, err
	}

	err = roles.InsertGroupOwner(s.E, body)
	if err != nil {
		return models.Bucket{}, err
	}

	err = roles.AddUserToOwners(s.E, body, claims)

	if err != nil {
		return models.Bucket{}, err
	}

	return body, nil
}

func (s BucketService) GetBucketList(u *models.UserClaims) []models.Bucket {
	var buckets []models.Bucket
	if !u.Valid() {
		zap.L().Warn(fmt.Sprintf("Invalid user claims %v", u.UserID.String()))
		return []models.Bucket{}
	}
	roles, err := s.E.GetImplicitRolesForUser(u.UserID.String(), configuration.DefaultDomain)
	if err != nil {
		zap.L().Warn(fmt.Sprintf("Error retrieving roles %v", u.UserID.String()))
		return []models.Bucket{}
	}

	var bucketPolicies []string
	for _, role := range roles {
		policies, _ := s.E.GetFilteredPolicy(0, configuration.DefaultDomain, role, "bucket", "", "read")
		for _, policy := range policies {
			bucketPolicies = append(bucketPolicies, policy[3])
		}
	}
	zap.L().Info("roles", zap.Any("bucketPolicies", bucketPolicies))
	_ = s.DB.Model(&models.Bucket{}).Where("id IN ?", bucketPolicies).Find(&buckets)
	return buckets
}

func (s BucketService) GetBucket(u *models.UserClaims, id uuid.UUID) (models.Bucket, error) {
	var bucket models.Bucket
	bucket.Files = []models.File{}

	result := s.DB.Where("id = ?", id).First(&bucket)
	if result.RowsAffected == 0 {
		return bucket, errors.New("bucket not found")
	} else {
		var files []models.File
		result = s.DB.Where("bucket_id = ?", id).Find(&files)
		if result.RowsAffected > 0 {
			bucket.Files = files
		}
		return bucket, nil
	}
}

func (s BucketService) UpdateBucket(u *models.UserClaims, id uuid.UUID, body models.Bucket) (models.Bucket, error) {
	bucket := models.Bucket{ID: id}
	result := s.DB.Model(&bucket).Updates(body)
	if result.RowsAffected == 0 {
		return bucket, errors.New("bucket not found")
	} else {
		return bucket, nil
	}
}

func (s BucketService) DeleteBucket(u *models.UserClaims, id uuid.UUID) error {
	result := s.DB.Where("id = ?", id).Delete(&models.Bucket{})
	if result.RowsAffected == 0 {
		return errors.New("bucket not found")
	} else {
		return nil
	}
}
