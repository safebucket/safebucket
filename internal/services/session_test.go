package services

import (
	"errors"
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/cache"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testRefreshTokenExpiry = 600 // minutes

func newSessionService(t *testing.T) (SessionService, *cache.MemoryCache) {
	t.Helper()
	mc := cache.NewMemoryCache()
	t.Cleanup(func() { mc.Close() })
	return SessionService{
		Cache:              mc,
		RefreshTokenExpiry: testRefreshTokenExpiry,
		ActivityLogger:     &MockActivityLogger{},
	}, mc
}

func TestListSessions_ReturnsSessions(t *testing.T) {
	svc, mc := newSessionService(t)
	userID := uuid.New()

	sid1 := uuid.New().String()
	sid2 := uuid.New().String()
	require.NoError(t, cache.CreateSession(mc, userID.String(), sid1))
	require.NoError(t, cache.CreateSession(mc, userID.String(), sid2))

	claims := models.UserClaims{
		UserID: userID,
		SID:    sid1,
	}

	resp, err := svc.ListSessions(zap.NewNop(), claims, uuid.UUIDs{userID})
	require.NoError(t, err)
	assert.Len(t, resp.Sessions, 2)

	ids := []string{resp.Sessions[0].ID, resp.Sessions[1].ID}
	assert.ElementsMatch(t, []string{sid1, sid2}, ids)

	for _, s := range resp.Sessions {
		assert.NotEmpty(t, s.CreatedAt)
	}
}

func TestListSessions_Empty(t *testing.T) {
	svc, _ := newSessionService(t)
	userID := uuid.New()

	claims := models.UserClaims{
		UserID: userID,
		SID:    uuid.New().String(),
	}

	resp, err := svc.ListSessions(zap.NewNop(), claims, uuid.UUIDs{userID})
	require.NoError(t, err)
	assert.Empty(t, resp.Sessions)
}

func TestListSessions_CurrentSessionFlag(t *testing.T) {
	svc, mc := newSessionService(t)
	userID := uuid.New()

	currentSID := uuid.New().String()
	otherSID := uuid.New().String()
	require.NoError(t, cache.CreateSession(mc, userID.String(), currentSID))
	require.NoError(t, cache.CreateSession(mc, userID.String(), otherSID))

	claims := models.UserClaims{
		UserID: userID,
		SID:    currentSID,
	}

	resp, err := svc.ListSessions(zap.NewNop(), claims, uuid.UUIDs{userID})
	require.NoError(t, err)

	for _, s := range resp.Sessions {
		if s.ID == currentSID {
			assert.True(t, s.IsCurrent)
		} else {
			assert.False(t, s.IsCurrent)
		}
	}
}

func TestRevokeSession_Success(t *testing.T) {
	svc, mc := newSessionService(t)
	userID := uuid.New()

	currentSID := uuid.New().String()
	targetSID := uuid.New().String()
	require.NoError(t, cache.CreateSession(mc, userID.String(), currentSID))
	require.NoError(t, cache.CreateSession(mc, userID.String(), targetSID))

	targetUUID, err := uuid.Parse(targetSID)
	require.NoError(t, err)

	claims := models.UserClaims{
		UserID: userID,
		SID:    currentSID,
	}

	err = svc.RevokeSession(zap.NewNop(), claims, uuid.UUIDs{userID, targetUUID})
	require.NoError(t, err)

	maxAge := time.Duration(testRefreshTokenExpiry) * time.Minute
	active, err := cache.IsSessionActive(mc, userID.String(), targetSID, maxAge)
	require.NoError(t, err)
	assert.False(t, active)
}

func TestRevokeSession_Self(t *testing.T) {
	svc, mc := newSessionService(t)
	userID := uuid.New()

	selfSID := uuid.New().String()
	require.NoError(t, cache.CreateSession(mc, userID.String(), selfSID))

	selfUUID, err := uuid.Parse(selfSID)
	require.NoError(t, err)

	claims := models.UserClaims{
		UserID: userID,
		SID:    selfSID,
	}

	err = svc.RevokeSession(zap.NewNop(), claims, uuid.UUIDs{userID, selfUUID})
	require.NoError(t, err)

	maxAge := time.Duration(testRefreshTokenExpiry) * time.Minute
	active, err := cache.IsSessionActive(mc, userID.String(), selfSID, maxAge)
	require.NoError(t, err)
	assert.False(t, active)
}

func TestRevokeSession_NotFound(t *testing.T) {
	svc, mc := newSessionService(t)
	userID := uuid.New()

	currentSID := uuid.New().String()
	require.NoError(t, cache.CreateSession(mc, userID.String(), currentSID))

	unknownUUID := uuid.New()

	claims := models.UserClaims{
		UserID: userID,
		SID:    currentSID,
	}

	err := svc.RevokeSession(zap.NewNop(), claims, uuid.UUIDs{userID, unknownUUID})
	require.Error(t, err)

	var apiErr *apierrors.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, 404, apiErr.Code)
	assert.Equal(t, "SESSION_NOT_FOUND", apiErr.Message)
}

func TestRevokeOtherSessions_Success(t *testing.T) {
	svc, mc := newSessionService(t)
	userID := uuid.New()

	currentSID := uuid.New().String()
	otherSID1 := uuid.New().String()
	otherSID2 := uuid.New().String()
	require.NoError(t, cache.CreateSession(mc, userID.String(), currentSID))
	require.NoError(t, cache.CreateSession(mc, userID.String(), otherSID1))
	require.NoError(t, cache.CreateSession(mc, userID.String(), otherSID2))

	claims := models.UserClaims{
		UserID: userID,
		SID:    currentSID,
	}

	err := svc.RevokeOtherSessions(zap.NewNop(), claims, uuid.UUIDs{userID})
	require.NoError(t, err)

	maxAge := time.Duration(testRefreshTokenExpiry) * time.Minute

	active, err := cache.IsSessionActive(mc, userID.String(), currentSID, maxAge)
	require.NoError(t, err)
	assert.True(t, active)

	active, err = cache.IsSessionActive(mc, userID.String(), otherSID1, maxAge)
	require.NoError(t, err)
	assert.False(t, active)

	active, err = cache.IsSessionActive(mc, userID.String(), otherSID2, maxAge)
	require.NoError(t, err)
	assert.False(t, active)
}
