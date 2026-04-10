package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testMaxAge = 10 * time.Minute

func TestCreateSession(t *testing.T) {
	mc := newTestCache(t)

	err := CreateSession(mc, "user1", "jti-abc")
	require.NoError(t, err)

	mc.mu.Lock()
	entries := mc.sortedSets[sessionKey("user1")]
	mc.mu.Unlock()

	require.Len(t, entries, 1)
	assert.Equal(t, "jti-abc", entries[0].member)
	assert.InDelta(t, float64(time.Now().Unix()), entries[0].score, 2)
}

func TestIsSessionActive_Active(t *testing.T) {
	mc := newTestCache(t)

	require.NoError(t, CreateSession(mc, "user1", "jti-1"))

	active, err := IsSessionActive(mc, "user1", "jti-1", testMaxAge)
	require.NoError(t, err)
	assert.True(t, active)
}

func TestIsSessionActive_Expired(t *testing.T) {
	mc := newTestCache(t)

	oldTS := float64(time.Now().Add(-testMaxAge - time.Minute).Unix())
	require.NoError(t, mc.ZAdd(sessionKey("user1"), oldTS, "jti-old"))

	active, err := IsSessionActive(mc, "user1", "jti-old", testMaxAge)
	require.NoError(t, err)
	assert.False(t, active)
}

func TestIsSessionActive_Revoked(t *testing.T) {
	mc := newTestCache(t)

	require.NoError(t, CreateSession(mc, "user1", "jti-1"))
	require.NoError(t, RevokeSession(mc, "user1", "jti-1"))

	active, err := IsSessionActive(mc, "user1", "jti-1", testMaxAge)
	require.NoError(t, err)
	assert.False(t, active)
}

func TestIsSessionActive_NotFound(t *testing.T) {
	mc := newTestCache(t)

	active, err := IsSessionActive(mc, "user1", "unknown-jti", testMaxAge)
	require.NoError(t, err)
	assert.False(t, active)
}

func TestListActiveSessions_ReturnsSessions(t *testing.T) {
	mc := newTestCache(t)

	require.NoError(t, CreateSession(mc, "user1", "jti-1"))
	require.NoError(t, CreateSession(mc, "user1", "jti-2"))

	sessions, err := ListActiveSessions(mc, "user1", testMaxAge)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)

	jtis := []string{sessions[0].SID, sessions[1].SID}
	assert.ElementsMatch(t, []string{"jti-1", "jti-2"}, jtis)

	for _, s := range sessions {
		assert.False(t, s.CreatedAt.IsZero())
	}
}

func TestListActiveSessions_PrunesExpired(t *testing.T) {
	mc := newTestCache(t)

	oldTS := float64(time.Now().Add(-testMaxAge - time.Minute).Unix())
	require.NoError(t, mc.ZAdd(sessionKey("user1"), oldTS, "jti-expired"))
	require.NoError(t, CreateSession(mc, "user1", "jti-active"))

	sessions, err := ListActiveSessions(mc, "user1", testMaxAge)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "jti-active", sessions[0].SID)
}

func TestListActiveSessions_ExcludesRevoked(t *testing.T) {
	mc := newTestCache(t)

	require.NoError(t, CreateSession(mc, "user1", "jti-1"))
	require.NoError(t, CreateSession(mc, "user1", "jti-2"))
	require.NoError(t, RevokeSession(mc, "user1", "jti-1"))

	sessions, err := ListActiveSessions(mc, "user1", testMaxAge)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "jti-2", sessions[0].SID)
}

func TestListActiveSessions_IsCurrent(t *testing.T) {
	mc := newTestCache(t)

	require.NoError(t, CreateSession(mc, "user1", "jti-current"))
	require.NoError(t, CreateSession(mc, "user1", "jti-other"))

	sessions, err := ListActiveSessions(mc, "user1", testMaxAge)
	require.NoError(t, err)

	for _, s := range sessions {
		if s.SID == "jti-current" {
			assert.Equal(t, "jti-current", s.SID)
		}
	}
}

func TestRevokeSession_SetsScoreToZero(t *testing.T) {
	mc := newTestCache(t)

	require.NoError(t, CreateSession(mc, "user1", "jti-1"))
	require.NoError(t, RevokeSession(mc, "user1", "jti-1"))

	active, err := IsSessionActive(mc, "user1", "jti-1", testMaxAge)
	require.NoError(t, err)
	assert.False(t, active)

	mc.mu.Lock()
	entries := mc.sortedSets[sessionKey("user1")]
	mc.mu.Unlock()
	require.Len(t, entries, 1)
	assert.Equal(t, float64(0), entries[0].score)
}

func TestRevokeOtherSessions_KeepsCurrent(t *testing.T) {
	mc := newTestCache(t)

	require.NoError(t, CreateSession(mc, "user1", "jti-current"))
	require.NoError(t, CreateSession(mc, "user1", "jti-other1"))
	require.NoError(t, CreateSession(mc, "user1", "jti-other2"))

	require.NoError(t, RevokeOtherSessions(mc, "user1", "jti-current", testMaxAge))

	active, err := IsSessionActive(mc, "user1", "jti-current", testMaxAge)
	require.NoError(t, err)
	assert.True(t, active)

	active, err = IsSessionActive(mc, "user1", "jti-other1", testMaxAge)
	require.NoError(t, err)
	assert.False(t, active)

	active, err = IsSessionActive(mc, "user1", "jti-other2", testMaxAge)
	require.NoError(t, err)
	assert.False(t, active)
}

func TestRevokeOtherSessions_EmptySet(t *testing.T) {
	mc := newTestCache(t)

	err := RevokeOtherSessions(mc, "user1", "jti-only", testMaxAge)
	require.NoError(t, err)
}

func TestRevokeAllSessions_DeletesSet(t *testing.T) {
	mc := newTestCache(t)

	require.NoError(t, CreateSession(mc, "user1", "jti-1"))
	require.NoError(t, CreateSession(mc, "user1", "jti-2"))

	require.NoError(t, RevokeAllSessions(mc, "user1"))

	sessions, err := ListActiveSessions(mc, "user1", testMaxAge)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestSessionCleanup_RemovesExpiredAndRevoked(t *testing.T) {
	mc := newTestCache(t)

	require.NoError(t, CreateSession(mc, "user1", "jti-active"))

	oldTS := float64(time.Now().Add(-testMaxAge - time.Minute).Unix())
	require.NoError(t, mc.ZAdd(sessionKey("user1"), oldTS, "jti-expired"))

	require.NoError(t, CreateSession(mc, "user1", "jti-revoked"))
	require.NoError(t, RevokeSession(mc, "user1", "jti-revoked"))

	keys, err := mc.ScanKeys("user:sessions:*", 100, 0)
	require.NoError(t, err)
	require.Len(t, keys, 1)

	cutoff := sessionCutoff(testMaxAge)
	for _, key := range keys {
		require.NoError(t, mc.ZRemRangeByScore(key, "-inf", cutoff))
	}
	sessions, err := ListActiveSessions(mc, "user1", testMaxAge)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "jti-active", sessions[0].SID)
}

func TestSessionCleanup_DeletesEmptyKeys(t *testing.T) {
	mc := newTestCache(t)
	oldTS := float64(time.Now().Add(-testMaxAge - time.Minute).Unix())
	require.NoError(t, mc.ZAdd(sessionKey("user1"), oldTS, "jti-expired"))

	keys, err := mc.ScanKeys("user:sessions:*", 100, 0)
	require.NoError(t, err)
	require.Len(t, keys, 1)

	cutoff := sessionCutoff(testMaxAge)
	for _, key := range keys {
		require.NoError(t, mc.ZRemRangeByScore(key, "-inf", cutoff))

		remaining, rangeErr := mc.ZRangeByScoreWithScores(key, "-inf", "+inf")
		require.NoError(t, rangeErr)
		if len(remaining) == 0 {
			require.NoError(t, mc.Del(key))
		}
	}

	keysAfter, err := mc.ScanKeys("user:sessions:*", 100, 0)
	require.NoError(t, err)
	assert.Empty(t, keysAfter)
}
