package services

import (
	"net/http"
	"strings"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
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

	r.Get("/", handlers.GetOneHandler(s.GetConversation))
	r.With(m.Validate[models.FileCommentCreateBody]).
		Post("/", handlers.CreateHandler(s.CreateComment))

	r.With(m.Validate[models.FileCommentUpdateBody]).
		Patch("/{id2}", handlers.BodyHandler(s.UpdateComment))
	r.Delete("/{id2}", handlers.DeleteHandler(s.DeleteComment))

	return r
}

func (s BucketFileCommentService) GetConversation(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) (models.FileConversationResponse, error) {
	bucketID, fileID := ids[0], ids[1]
	file, err := sql.GetFileByID(s.DB, bucketID, fileID)
	if err != nil {
		return models.FileConversationResponse{}, err
	}
	if err = sql.CheckFileCommentable(file); err != nil {
		return models.FileConversationResponse{}, err
	}

	group, err := s.viewerGroup(user, bucketID)
	if err != nil {
		return models.FileConversationResponse{}, err
	}

	members, err := sql.ListActiveMemberIDs(s.DB, bucketID)
	if err != nil {
		logger.Error("Failed to list active bucket members", zap.Error(err))
		return models.FileConversationResponse{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeFetchFailed,
		)
	}

	comments, err := sql.ListFileComments(s.DB, fileID)
	if err != nil {
		logger.Error("Failed to list file comments", zap.Error(err))
		return models.FileConversationResponse{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeFetchFailed,
		)
	}

	messages := make([]models.FileCommentResponse, 0, len(comments))
	for _, comment := range comments {
		mine := comment.UserID != nil && *comment.UserID == user.UserID
		message := models.FileCommentResponse{
			ID:        comment.ID,
			Body:      comment.Body,
			Version:   comment.Version,
			CanEdit:   mine,
			CanDelete: mine || group == models.GroupOwner,
			CreatedAt: comment.CreatedAt,
			UpdatedAt: comment.UpdatedAt,
		}
		if comment.User != nil {
			info := comment.User.ToActivity()
			message.User = &info
			message.UserActive = members[comment.User.ID]
		}
		messages = append(messages, message)
	}

	return models.FileConversationResponse{Messages: messages}, nil
}

func (s BucketFileCommentService) CreateComment(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FileCommentCreateBody,
) (models.FileCommentCreateResponse, error) {
	text, err := s.cleanBody(body.Body)
	if err != nil {
		return models.FileCommentCreateResponse{}, err
	}

	var comment models.FileComment
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		file, txErr := sql.LockFile(tx, ids[0], ids[1])
		if txErr != nil {
			return txErr
		}
		if txErr = sql.CheckFileCommentable(file); txErr != nil {
			return txErr
		}
		if file.CurrentVersionID == nil {
			return apierrors.New(http.StatusConflict, apierrors.CodeFileUploadInProgress)
		}

		version, txErr := sql.GetFileVersionByID(tx, file.ID, *file.CurrentVersionID)
		if txErr != nil {
			return txErr
		}

		comment = models.FileComment{
			FileID:  file.ID,
			UserID:  &user.UserID,
			Body:    text,
			Version: version.Version,
		}
		if txErr = tx.Omit("User").Create(&comment).Error; txErr != nil {
			logger.Error("Failed to create comment", zap.Error(txErr))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeCreateFailed)
		}
		return nil
	})
	if err != nil {
		return models.FileCommentCreateResponse{}, err
	}

	return models.FileCommentCreateResponse{ID: comment.ID}, nil
}

func (s BucketFileCommentService) UpdateComment(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FileCommentUpdateBody,
) error {
	text, err := s.cleanBody(body.Body)
	if err != nil {
		return err
	}

	comment, err := s.getComment(ids)
	if err != nil {
		return err
	}
	if comment.UserID == nil || *comment.UserID != user.UserID {
		return apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
	}
	if comment.Body == text {
		return nil
	}

	result := s.DB.Model(&models.FileComment{}).Where("id = ?", comment.ID).Update("body", text)
	if result.Error != nil {
		logger.Error("Failed to update comment", zap.Error(result.Error))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeUpdateFailed)
	}
	if result.RowsAffected == 0 {
		return apierrors.New(http.StatusNotFound, apierrors.CodeCommentNotFound)
	}

	return nil
}

func (s BucketFileCommentService) DeleteComment(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	comment, err := s.getComment(ids)
	if err != nil {
		return err
	}

	group, err := s.viewerGroup(user, ids[0])
	if err != nil {
		return err
	}
	mine := comment.UserID != nil && *comment.UserID == user.UserID
	if !mine && group != models.GroupOwner {
		return apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
	}

	if err = s.DB.Where("id = ?", comment.ID).Delete(&models.FileComment{}).Error; err != nil {
		logger.Error("Failed to delete comment", zap.Error(err))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeDeleteFailed)
	}

	return nil
}

// viewerGroup returns the caller's group in the bucket. Admins are treated as owners.
func (s BucketFileCommentService) viewerGroup(user models.UserClaims, bucketID uuid.UUID) (models.Group, error) {
	if user.Role == models.RoleAdmin {
		return models.GroupOwner, nil
	}

	membership, err := rbac.GetUserMembership(s.DB, user.UserID, bucketID)
	if err != nil {
		return "", err
	}
	if membership == nil {
		return "", apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
	}

	return membership.Group, nil
}

func (s BucketFileCommentService) getComment(ids uuid.UUIDs) (models.FileComment, error) {
	file, err := sql.GetFileByID(s.DB, ids[0], ids[1])
	if err != nil {
		return models.FileComment{}, err
	}
	if err = sql.CheckFileCommentable(file); err != nil {
		return models.FileComment{}, err
	}

	return sql.GetFileComment(s.DB, ids[1], ids[2])
}

func (s BucketFileCommentService) cleanBody(raw string) (string, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", apierrors.New(http.StatusBadRequest, apierrors.CodeFieldRequired)
	}
	if strings.ContainsRune(text, 0) {
		return "", apierrors.New(http.StatusBadRequest, apierrors.CodeInvalidValue)
	}
	return text, nil
}
