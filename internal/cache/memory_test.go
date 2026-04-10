package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCache(t *testing.T) *MemoryCache {
	t.Helper()
	mc := NewMemoryCache()
	t.Cleanup(func() { mc.Close() })
	return mc
}

func TestGet_Missing(t *testing.T) {
	mc := newTestCache(t)

	_, err := mc.Get("nonexistent")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestGet_AfterSetNX(t *testing.T) {
	mc := newTestCache(t)

	ok, err := mc.SetNX("k", "v", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)

	val, err := mc.Get("k")
	require.NoError(t, err)
	assert.Equal(t, "v", val)
}

func TestGet_Expired(t *testing.T) {
	mc := newTestCache(t)

	_, _ = mc.SetNX("k", "v", time.Millisecond)
	time.Sleep(2 * time.Millisecond)

	_, err := mc.Get("k")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestSetNX_AlreadyExists(t *testing.T) {
	mc := newTestCache(t)

	ok, _ := mc.SetNX("k", "first", time.Minute)
	assert.True(t, ok)

	ok, err := mc.SetNX("k", "second", time.Minute)
	require.NoError(t, err)
	assert.False(t, ok)

	val, _ := mc.Get("k")
	assert.Equal(t, "first", val)
}

func TestSetNX_AfterExpiry(t *testing.T) {
	mc := newTestCache(t)

	_, _ = mc.SetNX("k", "old", time.Millisecond)
	time.Sleep(2 * time.Millisecond)

	ok, err := mc.SetNX("k", "new", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)

	val, _ := mc.Get("k")
	assert.Equal(t, "new", val)
}

func TestDel(t *testing.T) {
	mc := newTestCache(t)

	_, _ = mc.SetNX("k", "v", time.Minute)

	err := mc.Del("k")
	require.NoError(t, err)

	_, err = mc.Get("k")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestDel_Missing(t *testing.T) {
	mc := newTestCache(t)
	assert.NoError(t, mc.Del("nonexistent"))
}

func TestIncr_NewKey(t *testing.T) {
	mc := newTestCache(t)

	val, err := mc.Incr("counter")
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
}

func TestIncr_Existing(t *testing.T) {
	mc := newTestCache(t)

	_, _ = mc.Incr("counter")
	_, _ = mc.Incr("counter")
	val, err := mc.Incr("counter")

	require.NoError(t, err)
	assert.Equal(t, int64(3), val)
}

func TestIncr_ExpiredKey(t *testing.T) {
	mc := newTestCache(t)

	_, _ = mc.Incr("counter")
	_ = mc.Expire("counter", time.Millisecond)
	time.Sleep(2 * time.Millisecond)

	val, err := mc.Incr("counter")
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
}

func TestIncr_NonNumericValue(t *testing.T) {
	mc := newTestCache(t)

	mc.mu.Lock()
	mc.data["k"] = entry{value: "not-a-number"}
	mc.mu.Unlock()

	_, err := mc.Incr("k")
	assert.Error(t, err)
}

func TestExpire_SetsExpiry(t *testing.T) {
	mc := newTestCache(t)

	_, _ = mc.Incr("k")

	err := mc.Expire("k", 50*time.Millisecond)
	require.NoError(t, err)

	val, _ := mc.Get("k")
	assert.Equal(t, "1", val)

	time.Sleep(60 * time.Millisecond)

	_, err = mc.Get("k")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestExpire_MissingKey(t *testing.T) {
	mc := newTestCache(t)
	assert.NoError(t, mc.Expire("nonexistent", time.Minute))
}

func TestTTL_MissingKey(t *testing.T) {
	mc := newTestCache(t)

	ttl, err := mc.TTL("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, -2*time.Second, ttl)
}

func TestTTL_NoExpiry(t *testing.T) {
	mc := newTestCache(t)

	_, _ = mc.Incr("k") // creates key with no TTL

	ttl, err := mc.TTL("k")
	require.NoError(t, err)
	assert.Equal(t, -1*time.Second, ttl)
}

func TestTTL_WithExpiry(t *testing.T) {
	mc := newTestCache(t)

	_, _ = mc.SetNX("k", "v", time.Minute)

	ttl, err := mc.TTL("k")
	require.NoError(t, err)
	assert.True(t, ttl > 50*time.Second && ttl <= time.Minute)
}

func TestZAdd_NewMember(t *testing.T) {
	mc := newTestCache(t)

	require.NoError(t, mc.ZAdd("zset", 1.0, "a"))
	require.NoError(t, mc.ZAdd("zset", 2.0, "b"))

	mc.mu.Lock()
	assert.Len(t, mc.sortedSets["zset"], 2)
	mc.mu.Unlock()
}

func TestZAdd_UpdateScore(t *testing.T) {
	mc := newTestCache(t)

	_ = mc.ZAdd("zset", 1.0, "a")
	_ = mc.ZAdd("zset", 5.0, "a")

	mc.mu.Lock()
	assert.Len(t, mc.sortedSets["zset"], 1)
	assert.Equal(t, 5.0, mc.sortedSets["zset"][0].score)
	mc.mu.Unlock()
}

func TestZRemRangeByScore_RemovesInRange(t *testing.T) {
	mc := newTestCache(t)

	_ = mc.ZAdd("zset", 1.0, "a")
	_ = mc.ZAdd("zset", 5.0, "b")
	_ = mc.ZAdd("zset", 10.0, "c")

	err := mc.ZRemRangeByScore("zset", "-inf", "5")
	require.NoError(t, err)

	mc.mu.Lock()
	assert.Len(t, mc.sortedSets["zset"], 1)
	assert.Equal(t, "c", mc.sortedSets["zset"][0].member)
	mc.mu.Unlock()
}

func TestZRemRangeByScore_RemovesAll(t *testing.T) {
	mc := newTestCache(t)

	_ = mc.ZAdd("zset", 1.0, "a")
	_ = mc.ZAdd("zset", 2.0, "b")

	err := mc.ZRemRangeByScore("zset", "-inf", "+inf")
	require.NoError(t, err)

	mc.mu.Lock()
	_, exists := mc.sortedSets["zset"]
	mc.mu.Unlock()
	assert.False(t, exists)
}

func TestZRemRangeByScore_MissingKey(t *testing.T) {
	mc := newTestCache(t)
	assert.NoError(t, mc.ZRemRangeByScore("nonexistent", "0", "10"))
}

func TestZScore_Existing(t *testing.T) {
	mc := newTestCache(t)

	_ = mc.ZAdd("zset", 42.5, "a")
	_ = mc.ZAdd("zset", 99.0, "b")

	score, err := mc.ZScore("zset", "a")
	require.NoError(t, err)
	assert.Equal(t, 42.5, score)

	score, err = mc.ZScore("zset", "b")
	require.NoError(t, err)
	assert.Equal(t, 99.0, score)
}

func TestZScore_MissingMember(t *testing.T) {
	mc := newTestCache(t)

	_ = mc.ZAdd("zset", 1.0, "a")

	_, err := mc.ZScore("zset", "nonexistent")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestZScore_MissingKey(t *testing.T) {
	mc := newTestCache(t)

	_, err := mc.ZScore("nonexistent", "a")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestZRangeByScoreWithScores_ReturnsInRange(t *testing.T) {
	mc := newTestCache(t)

	_ = mc.ZAdd("zset", 1.0, "a")
	_ = mc.ZAdd("zset", 5.0, "b")
	_ = mc.ZAdd("zset", 10.0, "c")

	entries, err := mc.ZRangeByScoreWithScores("zset", "1", "5")
	require.NoError(t, err)
	assert.ElementsMatch(t, []ZScoreEntry{
		{Member: "a", Score: 1.0},
		{Member: "b", Score: 5.0},
	}, entries)
}

func TestZRangeByScoreWithScores_InfBounds(t *testing.T) {
	mc := newTestCache(t)

	_ = mc.ZAdd("zset", 1.0, "a")
	_ = mc.ZAdd("zset", 5.0, "b")
	_ = mc.ZAdd("zset", 10.0, "c")

	entries, err := mc.ZRangeByScoreWithScores("zset", "-inf", "+inf")
	require.NoError(t, err)
	assert.ElementsMatch(t, []ZScoreEntry{
		{Member: "a", Score: 1.0},
		{Member: "b", Score: 5.0},
		{Member: "c", Score: 10.0},
	}, entries)
}

func TestZRangeByScoreWithScores_MissingKey(t *testing.T) {
	mc := newTestCache(t)

	entries, err := mc.ZRangeByScoreWithScores("nonexistent", "0", "10")
	require.NoError(t, err)
	assert.Nil(t, entries)
}

func TestZRangeByScoreWithScores_NoMatches(t *testing.T) {
	mc := newTestCache(t)

	_ = mc.ZAdd("zset", 1.0, "a")
	_ = mc.ZAdd("zset", 2.0, "b")

	entries, err := mc.ZRangeByScoreWithScores("zset", "5", "10")
	require.NoError(t, err)
	assert.Nil(t, entries)
}

func TestScanKeys_MatchesSortedSets(t *testing.T) {
	mc := newTestCache(t)

	_ = mc.ZAdd("user:sessions:abc", 1.0, "s1")
	_ = mc.ZAdd("user:sessions:def", 2.0, "s2")
	_ = mc.ZAdd("other:key", 3.0, "s3")

	keys, err := mc.ScanKeys("user:sessions:*", 100, 0)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.ElementsMatch(t, []string{"user:sessions:abc", "user:sessions:def"}, keys)
}

func TestScanKeys_MatchesDataKeys(t *testing.T) {
	mc := newTestCache(t)

	_, _ = mc.SetNX("app:ratelimit:user1", "5", time.Minute)
	_, _ = mc.SetNX("app:ratelimit:user2", "3", time.Minute)
	_, _ = mc.SetNX("other:key", "1", time.Minute)

	keys, err := mc.ScanKeys("app:ratelimit:*", 100, 0)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.ElementsMatch(t, []string{"app:ratelimit:user1", "app:ratelimit:user2"}, keys)
}

func TestScanKeys_NoMatches(t *testing.T) {
	mc := newTestCache(t)

	_ = mc.ZAdd("user:sessions:abc", 1.0, "s1")

	keys, err := mc.ScanKeys("nonexistent:*", 100, 0)
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestScanKeys_NoDuplicates(t *testing.T) {
	mc := newTestCache(t)
	_, _ = mc.SetNX("user:sessions:abc", "val", time.Minute)
	_ = mc.ZAdd("user:sessions:abc", 1.0, "s1")

	keys, err := mc.ScanKeys("user:sessions:*", 100, 0)
	require.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, "user:sessions:abc", keys[0])
}
