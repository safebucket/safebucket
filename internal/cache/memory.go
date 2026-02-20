package cache

import (
	"math"
	"strconv"
	"sync"
	"time"
)

type entry struct {
	value     string
	expiresAt time.Time
}

func (e entry) isExpired() bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}

type sortedSetEntry struct {
	score  float64
	member string
}

type MemoryCache struct {
	mu         sync.Mutex
	data       map[string]entry
	sortedSets map[string][]sortedSetEntry
	stopCh     chan struct{}
}

func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		data:       make(map[string]entry),
		sortedSets: make(map[string][]sortedSetEntry),
		stopCh:     make(chan struct{}),
	}
	go mc.cleanupLoop()
	return mc
}

func (m *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()
			for k, e := range m.data {
				if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
					delete(m.data, k)
				}
			}
			m.mu.Unlock()
		}
	}
}

func (m *MemoryCache) Get(key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.data[key]
	if !ok || e.isExpired() {
		delete(m.data, key)
		return "", ErrKeyNotFound
	}
	return e.value, nil
}

func (m *MemoryCache) SetNX(key string, value string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e, ok := m.data[key]; ok && !e.isExpired() {
		return false, nil
	}

	m.data[key] = entry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return true, nil
}

func (m *MemoryCache) Del(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	return nil
}

func (m *MemoryCache) Incr(key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.data[key]
	if !ok || e.isExpired() {
		m.data[key] = entry{value: "1"}
		return 1, nil
	}

	val, err := strconv.ParseInt(e.value, 10, 64)
	if err != nil {
		return 0, err
	}

	val++
	e.value = strconv.FormatInt(val, 10)
	m.data[key] = e
	return val, nil
}

func (m *MemoryCache) Expire(key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.data[key]
	if !ok || e.isExpired() {
		return nil
	}

	e.expiresAt = time.Now().Add(ttl)
	m.data[key] = e
	return nil
}

func (m *MemoryCache) TTL(key string) (time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.data[key]
	if !ok || e.isExpired() {
		return -2 * time.Second, nil
	}

	if e.expiresAt.IsZero() {
		return -1 * time.Second, nil
	}

	remaining := time.Until(e.expiresAt)
	if remaining < 0 {
		delete(m.data, key)
		return -2 * time.Second, nil
	}
	return remaining, nil
}

func (m *MemoryCache) ZAdd(key string, score float64, member string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries := m.sortedSets[key]
	for i, e := range entries {
		if e.member == member {
			entries[i].score = score
			m.sortedSets[key] = entries
			return nil
		}
	}

	m.sortedSets[key] = append(entries, sortedSetEntry{score: score, member: member})
	return nil
}

func (m *MemoryCache) ZRemRangeByScore(key string, minScore string, maxScore string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, ok := m.sortedSets[key]
	if !ok {
		return nil
	}

	lo := parseScore(minScore)
	hi := parseScore(maxScore)

	filtered := entries[:0]
	for _, e := range entries {
		if e.score < lo || e.score > hi {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) == 0 {
		delete(m.sortedSets, key)
	} else {
		m.sortedSets[key] = filtered
	}
	return nil
}

func parseScore(s string) float64 {
	switch s {
	case "-inf":
		return math.Inf(-1)
	case "+inf", "inf":
		return math.Inf(1)
	default:
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}
}

func (m *MemoryCache) Close() {
	close(m.stopCh)
}
