package services

import (
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/cache"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SessionService struct {
	Cache              cache.ICache
	RefreshTokenExpiry int
	ActivityLogger     activity.IActivityLogger
}

func (s SessionService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeSelfOrAdmin(0)).
		Get("/", handlers.GetOneHandler(s.ListSessions))
	r.With(m.AuthorizeSelfOrAdmin(0)).
		Delete("/", handlers.DeleteHandler(s.RevokeOtherSessions))

	r.Route("/{id1}", func(r chi.Router) {
		r.With(m.AuthorizeSelfOrAdmin(0)).
			Delete("/", handlers.DeleteHandler(s.RevokeSession))
	})

	return r
}

func (s SessionService) maxAge() time.Duration {
	return time.Duration(s.RefreshTokenExpiry) * time.Minute
}

type SessionResponse struct {
	ID        string `json:"id"`
	IsCurrent bool   `json:"is_current"`
	CreatedAt string `json:"created_at"`
}

type SessionListResponse struct {
	Sessions []SessionResponse `json:"sessions"`
}

func (s SessionService) ListSessions(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
) (SessionListResponse, error) {
	userID := ids[0]

	sessions, err := cache.ListActiveSessions(s.Cache, userID.String(), s.maxAge())
	if err != nil {
		logger.Error("Failed to list sessions", zap.Error(err))
		return SessionListResponse{}, apierrors.ErrInternalServer
	}

	resp := SessionListResponse{Sessions: make([]SessionResponse, 0, len(sessions))}
	for _, sess := range sessions {
		resp.Sessions = append(resp.Sessions, SessionResponse{
			ID:        sess.SID,
			IsCurrent: sess.SID == claims.SID,
			CreatedAt: sess.CreatedAt.Format(time.RFC3339),
		})
	}
	return resp, nil
}

func (s SessionService) RevokeOtherSessions(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
) error {
	userID := ids[0]

	if err := cache.RevokeOtherSessions(
		s.Cache, userID.String(), claims.SID, s.maxAge(),
	); err != nil {
		logger.Error("Failed to revoke other sessions", zap.Error(err))
		return apierrors.ErrInternalServer
	}

	action := models.Activity{
		Message: activity.OtherSessionsRevoked,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      activity.OtherSessionsRevoked,
			"user_id":     userID.String(),
			"object_type": "session",
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log other sessions revocation", zap.Error(logErr))
	}

	return nil
}

func (s SessionService) RevokeSession(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) error {
	userID := ids[0]
	sessionSID := ids[1].String()

	active, err := cache.IsSessionActive(s.Cache, userID.String(), sessionSID, s.maxAge())
	if err != nil {
		logger.Error("Failed to check session", zap.Error(err))
		return apierrors.ErrInternalServer
	}

	if !active {
		return apierrors.NewAPIError(404, "SESSION_NOT_FOUND")
	}

	if err = cache.RevokeSession(s.Cache, userID.String(), sessionSID); err != nil {
		logger.Error("Failed to revoke session", zap.Error(err))
		return apierrors.ErrInternalServer
	}

	action := models.Activity{
		Message: activity.SessionRevoked,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      activity.SessionRevoked,
			"user_id":     userID.String(),
			"session_id":  sessionSID,
			"object_type": "session",
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log session revocation", zap.Error(logErr))
	}

	return nil
}
