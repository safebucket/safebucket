package services

import (
	"net/http"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/events"
	"github.com/safebucket/safebucket/internal/fileversions"
	"github.com/safebucket/safebucket/internal/handlers"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/messaging"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PublicShareService struct {
	DB                    *gorm.DB
	Storage               storage.IStorage
	ActivityLogger        activity.IActivityLogger
	Publisher             messaging.IPublisher
	TokenSecret           string
	CookieSecureForce     bool
	AllowRedirectDownload bool
	Versions              fileversions.Manager
}

func (s PublicShareService) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/{shareId}", func(r chi.Router) {
		r.Use(m.ValidateShareAccess(s.DB))

		r.With(m.Validate[models.ShareAuthBody]).
			Post("/auth", handlers.ShareAuthHandler(s.CookieSecureForce, s.AuthenticateShare))

		r.Group(func(r chi.Router) {
			r.Use(m.ValidateShareToken(s.TokenSecret))

			r.Get("/", handlers.ShareGetOneHandler(s.ListShareItems))
			r.Get("/download", handlers.ShareDownloadRedirectHandler(s.DownloadSingleShareFile))
			r.With(m.ValidateQuery[models.FileDownloadQuery]).
				Get("/files/{id1}/url", handlers.ShareGetOneWithQueryHandler(s.DownloadShareFile))
			r.Get("/files/{id1}/download", handlers.ShareDownloadRedirectHandler(s.DownloadShareFileRedirect))
			r.With(m.Validate[models.ShareUploadBody]).
				Post("/files", handlers.ShareCreateHandler(s.UploadShareFile))
			r.Patch("/files/{id1}", handlers.ShareActionHandler(s.ConfirmShareUpload))
			r.Patch("/files/{id1}/versions/{id2}", handlers.ShareActionHandler(s.ConfirmShareVersion))
			r.Delete("/files/{id1}/versions/{id2}", handlers.ShareActionHandler(s.CancelShareVersion))
		})
	})

	return r
}

func (s PublicShareService) AuthenticateShare(
	isSecure bool,
	logger *zap.Logger,
	share models.Share,
	_ uuid.UUIDs,
	body models.ShareAuthBody,
) (handlers.AuthFlowResult, error) {
	if share.HashedPassword == "" {
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusBadRequest, apierrors.CodeShareNotPasswordProtected)
	}

	match, err := argon2id.ComparePasswordAndHash(body.Password, share.HashedPassword)
	if err != nil || !match {
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusUnauthorized, apierrors.CodeSharePasswordInvalid)
	}

	token, err := h.NewShareAccessToken(s.TokenSecret, share.ID)
	if err != nil {
		logger.Error("Failed to create share access token", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	publicID := share.ID.String()
	if share.CustomID != nil {
		publicID = *share.CustomID
	}

	return handlers.AuthFlowResult{
		Status:  http.StatusOK,
		Body:    struct{}{},
		Cookies: handlers.BuildShareCookie(isSecure, publicID, token),
	}, nil
}

func (s PublicShareService) ListShareItems(
	logger *zap.Logger,
	share models.Share,
	_ uuid.UUIDs,
) (models.PublicShareResponse, error) {
	result := s.DB.Model(&models.Share{}).
		Where("id = ? AND (max_views IS NULL OR current_views < max_views)", share.ID).
		UpdateColumn("current_views", gorm.Expr("current_views + 1"))
	if result.Error != nil {
		logger.Error("Failed to increment share views", zap.Error(result.Error))
		return models.PublicShareResponse{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	if result.RowsAffected == 0 {
		return models.PublicShareResponse{}, apierrors.New(http.StatusForbidden, apierrors.CodeShareMaxViewsReached)
	}

	var sharedBy models.User
	if err := s.DB.Unscoped().
		Select("id", "first_name", "last_name", "email").
		Find(&sharedBy, share.CreatedBy).Error; err != nil {
		logger.Error("Failed to load share creator", zap.Error(err))
		return models.PublicShareResponse{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	response := models.PublicShareResponse{
		ID:                share.ID,
		CustomID:          share.CustomID,
		Name:              share.Name,
		Type:              share.Type,
		FolderID:          share.FolderID,
		AllowUpload:       share.AllowUpload,
		PasswordProtected: share.HashedPassword != "",
		MaxUploadSize:     share.MaxUploadSize,
		MaxUploads:        share.MaxUploads,
		CurrentUploads:    share.CurrentUploads,
		ExpiresAt:         share.ExpiresAt,
		MaxViews:          share.MaxViews,
		CurrentViews:      share.CurrentViews + 1,
		SharedBy: models.UserInfo{
			ID:        sharedBy.ID,
			FirstName: sharedBy.FirstName,
			LastName:  sharedBy.LastName,
			Email:     sharedBy.Email,
		},
		Files:   []models.File{},
		Folders: []models.Folder{},
	}

	now := time.Now()

	switch share.Type {
	case models.ShareTypeFiles:
		var files []models.File
		s.DB.Joins("JOIN share_files ON share_files.file_id = files.id").
			Where("share_files.share_id = ?", share.ID).
			Where("files.status = ?", models.FileStatusUploaded).
			Where("files.expires_at IS NULL OR files.expires_at > ?", now).
			Find(&files)
		response.Files = files

	case models.ShareTypeFolder:
		folders, err := sql.GetFolderSubtree(s.DB, share.BucketID, *share.FolderID)
		if err != nil {
			logger.Error("Failed to list shared folder subtree", zap.Error(err))
			return models.PublicShareResponse{}, apierrors.New(
				http.StatusInternalServerError,
				apierrors.CodeInternalServerError,
			)
		}
		response.Folders = folders

		folderIDs := make(uuid.UUIDs, 1, len(folders)+1)
		folderIDs[0] = *share.FolderID
		for _, folder := range folders {
			folderIDs = append(folderIDs, folder.ID)
		}

		var files []models.File
		s.DB.Where(
			"bucket_id = ? AND folder_id IN ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)",
			share.BucketID, folderIDs, models.FileStatusUploaded, now,
		).Find(&files)
		response.Files = files

	case models.ShareTypeBucket:
		var files []models.File
		s.DB.Where(
			"bucket_id = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)",
			share.BucketID, models.FileStatusUploaded, now,
		).Find(&files)
		response.Files = files

		var folders []models.Folder
		s.DB.Where("bucket_id = ? AND status = ?", share.BucketID, models.FolderStatusCreated).Find(&folders)
		response.Folders = folders
	}

	return response, nil
}

func (s PublicShareService) DownloadShareFile(
	logger *zap.Logger,
	share models.Share,
	ids uuid.UUIDs,
	query models.FileDownloadQuery,
) (models.FileDownloadResponse, error) {
	fileID := ids[1]

	file, err := h.GetShareFile(s.DB, share, fileID)
	if err != nil {
		return models.FileDownloadResponse{}, err
	}

	var inlineContentType string
	if query.Context == "preview" {
		inlineContentType = h.PreviewMimeFromExtension(file.Extension)
	}

	url, err := s.Storage.PresignedGetObject(
		storage.VersionObjectKey(share.BucketID, file.ContentVersionID()),
		storage.GetObjectOptions{
			InlineContentType: inlineContentType,
			DownloadFilename:  file.Name,
		},
	)
	if err != nil {
		logger.Error("Generate presigned URL failed", zap.Error(err))
		return models.FileDownloadResponse{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	if activityErr := s.ActivityLogger.Send(models.Activity{
		Message: activity.ShareFileDownloaded,
		Object:  file.ToActivity(),
		Filter: activity.NewLogFilter(models.ActivityFields{
			Action:     rbac.ActionDownload.String(),
			ObjectType: rbac.ResourceFile.String(),
			BucketID:   share.BucketID.String(),
			FileID:     fileID.String(),
			ShareID:    share.ID.String(),
		}),
	}); activityErr != nil {
		logger.Error("Failed to log share download activity", zap.Error(activityErr))
		return models.FileDownloadResponse{}, activityErr
	}

	return models.FileDownloadResponse{
		ID:  file.ID.String(),
		URL: url,
	}, nil
}

func (s PublicShareService) DownloadSingleShareFile(
	logger *zap.Logger,
	share models.Share,
	_ uuid.UUIDs,
) (models.FileDownloadResponse, error) {
	if !s.AllowRedirectDownload {
		return models.FileDownloadResponse{}, apierrors.New(
			http.StatusForbidden, apierrors.CodeRedirectDownloadDisabled,
		)
	}

	if share.Type != models.ShareTypeFiles {
		return models.FileDownloadResponse{}, apierrors.New(http.StatusConflict, apierrors.CodeShareNotSingleFile)
	}

	now := time.Now()

	var files []models.File
	s.DB.Joins("JOIN share_files ON share_files.file_id = files.id").
		Where("share_files.share_id = ?", share.ID).
		Where("files.status = ?", models.FileStatusUploaded).
		Where("files.expires_at IS NULL OR files.expires_at > ?", now).
		Find(&files)

	if len(files) != 1 {
		return models.FileDownloadResponse{}, apierrors.New(http.StatusConflict, apierrors.CodeShareNotSingleFile)
	}

	ids := uuid.UUIDs{share.ID, files[0].ID}

	return s.DownloadShareFile(logger, share, ids, models.FileDownloadQuery{Context: "download"})
}

func (s PublicShareService) DownloadShareFileRedirect(
	logger *zap.Logger,
	share models.Share,
	ids uuid.UUIDs,
) (models.FileDownloadResponse, error) {
	if !s.AllowRedirectDownload {
		return models.FileDownloadResponse{}, apierrors.New(
			http.StatusForbidden, apierrors.CodeRedirectDownloadDisabled,
		)
	}

	return s.DownloadShareFile(logger, share, ids, models.FileDownloadQuery{Context: "download"})
}

func (s PublicShareService) UploadShareFile(
	logger *zap.Logger,
	share models.Share,
	_ uuid.UUIDs,
	body models.ShareUploadBody,
) (models.FileUploadResponse, error) {
	if !share.AllowUpload {
		return models.FileUploadResponse{}, apierrors.New(http.StatusForbidden, apierrors.CodeShareUploadNotAllowed)
	}

	if share.MaxUploadSize != nil && body.Size > *share.MaxUploadSize {
		return models.FileUploadResponse{}, apierrors.New(
			http.StatusBadRequest,
			apierrors.CodeShareUploadSizeExceeded,
		)
	}

	if share.MaxUploads != nil && share.CurrentUploads >= *share.MaxUploads {
		return models.FileUploadResponse{}, apierrors.New(http.StatusForbidden, apierrors.CodeMaxUploadsReached)
	}

	var folderID *uuid.UUID
	switch share.Type {
	case models.ShareTypeFiles:
		folderID = nil
	case models.ShareTypeFolder:
		folderID = share.FolderID
		if body.FolderID != nil {
			if share.FolderID == nil || !h.IsFolderDescendant(s.DB, body.FolderID, *share.FolderID) {
				return models.FileUploadResponse{}, apierrors.New(
					http.StatusForbidden,
					apierrors.CodeShareFileNotInShare,
				)
			}
			folderID = body.FolderID
		}
	case models.ShareTypeBucket:
		folderID = body.FolderID
	}

	return s.Versions.Start(logger, share.BucketID, models.FileUploadBody{
		Name: body.Name, Size: int(body.Size), FolderID: folderID,
	}, nil, &share)
}

func (s PublicShareService) ConfirmShareUpload(logger *zap.Logger, share models.Share, ids uuid.UUIDs) error {
	return s.ConfirmShareVersion(logger, share, uuid.UUIDs{ids[0], ids[1], uuid.Nil})
}

func (s PublicShareService) ConfirmShareVersion(logger *zap.Logger, share models.Share, ids uuid.UUIDs) error {
	completed, err := s.Versions.Complete(logger, share.BucketID, ids[1], ids[2], &share, false)
	if err == nil && completed.Changed {
		events.NotifyFileVersionUploaded(logger, s.DB, s.ActivityLogger, s.Publisher, completed)
	}
	return err
}

func (s PublicShareService) CancelShareVersion(logger *zap.Logger, share models.Share, ids uuid.UUIDs) error {
	_, _, err := s.Versions.Delete(logger, share.BucketID, ids[1], ids[2], &share)
	return err
}
