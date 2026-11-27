package services

import (
	"net/http"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BucketFileCommentService struct {
	DB *gorm.DB
}

func (s BucketFileCommentService) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(m.AuthorizeGroup(s.DB, models.GroupViewer, 0))

	r.Get("/", handlers.GetListHandler(s.GetComments))
	r.With(m.Validate[models.FileCommentBody]).
		Post("/", handlers.CreateHandler(s.CreateComment))

	r.With(m.Validate[models.FileCommentBody]).
		Patch("/{id2}", handlers.BodyHandler(s.UpdateComment))
	r.Delete("/{id2}", handlers.DeleteHandler(s.DeleteComment))

	return r
}

func (s BucketFileCommentService) GetComments(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) []models.FileCommentResponse {
	file, err := s.getFile(ids[0], ids[1])
	if err != nil {
		logger.Warn("Failed to load file for comment listing", zap.Error(err))
		return []models.FileCommentResponse{}
	}

	comments, err := sql.ListFileComments(s.DB, file.ID)
	if err != nil {
		logger.Error("Failed to list file comments", zap.Error(err))
		return []models.FileCommentResponse{}
	}

	response := make([]models.FileCommentResponse, 0, len(comments))
	for _, comment := range comments {
		var author *models.UserInfo
		if comment.User != nil {
			info := comment.User.ToActivity()
			author = &info
		}
		response = append(response, models.FileCommentResponse{
			ID:        comment.ID,
			User:      author,
			Body:      comment.Body,
			Version:   comment.Version,
			CreatedAt: comment.CreatedAt,
			UpdatedAt: comment.UpdatedAt,
		})
	}

	return response
}

func (s BucketFileCommentService) CreateComment(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FileCommentBody,
) (models.FileCommentResponse, error) {
	file, err := s.getFile(ids[0], ids[1])
	if err != nil {
		return models.FileCommentResponse{}, err
	}
	if file.CurrentVersionID == nil {
		return models.FileCommentResponse{}, apierrors.New(http.StatusConflict, apierrors.CodeFileUploadInProgress)
	}

	version, err := sql.GetFileVersionByID(s.DB, file.ID, *file.CurrentVersionID)
	if err != nil {
		return models.FileCommentResponse{}, err
	}

	author, err := sql.GetUserByID(s.DB, user.UserID)
	if err != nil {
		return models.FileCommentResponse{}, err
	}

	comment := models.FileComment{
		FileID:  file.ID,
		UserID:  &user.UserID,
		Body:    body.Body,
		Version: version.Version,
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Omit("User").Create(&comment).Error; createErr != nil {
			logger.Error("Failed to create comment", zap.Error(createErr))
			return createErr
		}

		if logErr := outbox.AddActivities(tx, models.Activity{
			Message: activity.FileCommentCreated,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionCreate.String(),
				BucketID:   file.BucketID.String(),
				FileID:     file.ID.String(),
				CommentID:  comment.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
				UserID:     user.UserID.String(),
			}),
		}); logErr != nil {
			logger.Error("Failed to log comment creation activity", zap.Error(logErr))
			return logErr
		}

		return nil
	})
	if err != nil {
		return models.FileCommentResponse{}, apierrors.New(http.StatusInternalServerError, apierrors.CodeCreateFailed)
	}

	info := author.ToActivity()
	return models.FileCommentResponse{
		ID:        comment.ID,
		User:      &info,
		Body:      comment.Body,
		Version:   comment.Version,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}, nil
}

func (s BucketFileCommentService) UpdateComment(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FileCommentBody,
) error {
	file, comment, err := s.getComment(ids)
	if err != nil {
		return err
	}
	if comment.UserID == nil || *comment.UserID != user.UserID {
		return apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
	}

	previousBody := comment.Body
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if updateErr := tx.Model(&comment).Update("body", body.Body).Error; updateErr != nil {
			logger.Error("Failed to update comment", zap.Error(updateErr))
			return updateErr
		}

		if logErr := outbox.AddActivities(tx, models.Activity{
			Message: activity.FileCommentUpdated,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionUpdate.String(),
				BucketID:   file.BucketID.String(),
				FileID:     file.ID.String(),
				CommentID:  comment.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
				UserID:     user.UserID.String(),
			}),
			Changes: map[string]models.ActivityChange{
				"body": {From: previousBody, To: body.Body},
			},
		}); logErr != nil {
			logger.Error("Failed to log comment update activity", zap.Error(logErr))
			return logErr
		}

		return nil
	})
	if err != nil {
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeUpdateFailed)
	}

	return nil
}

func (s BucketFileCommentService) DeleteComment(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	file, comment, err := s.getComment(ids)
	if err != nil {
		return err
	}

	mine := comment.UserID != nil && *comment.UserID == user.UserID
	if !mine && user.Role != models.RoleAdmin {
		owner, accessErr := rbac.HasBucketAccess(s.DB, user.UserID, ids[0], models.GroupOwner)
		if accessErr != nil {
			logger.Error("Failed to check bucket access", zap.Error(accessErr))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeDeleteFailed)
		}
		if !owner {
			return apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
		}
	}

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if deleteErr := tx.Where("id = ?", comment.ID).Delete(&models.FileComment{}).Error; deleteErr != nil {
			logger.Error("Failed to delete comment", zap.Error(deleteErr))
			return deleteErr
		}

		if logErr := outbox.AddActivities(tx, models.Activity{
			Message: activity.FileCommentDeleted,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionDelete.String(),
				BucketID:   file.BucketID.String(),
				FileID:     file.ID.String(),
				CommentID:  comment.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
				UserID:     user.UserID.String(),
			}),
			Changes: map[string]models.ActivityChange{
				"body": {From: comment.Body},
			},
		}); logErr != nil {
			logger.Error("Failed to log comment deletion activity", zap.Error(logErr))
			return logErr
		}

		return nil
	})
	if err != nil {
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeDeleteFailed)
	}

	return nil
}

func (s BucketFileCommentService) getFile(bucketID, fileID uuid.UUID) (models.File, error) {
	file, err := sql.GetFileByID(s.DB.Unscoped(), bucketID, fileID)
	if err != nil {
		return models.File{}, err
	}
	if file.ExpiresAt != nil && !file.ExpiresAt.After(time.Now()) {
		return models.File{}, apierrors.New(http.StatusForbidden, apierrors.CodeFileExpired)
	}

	return file, nil
}

func (s BucketFileCommentService) getComment(ids uuid.UUIDs) (models.File, models.FileComment, error) {
	file, err := s.getFile(ids[0], ids[1])
	if err != nil {
		return models.File{}, models.FileComment{}, err
	}

	comment, err := sql.GetFileComment(s.DB, file.ID, ids[2])
	if err != nil {
		return models.File{}, models.FileComment{}, err
	}

	return file, comment, nil
}
