package helpers

import (
	"errors"
	"testing"

	"github.com/safebucket/safebucket/internal/storage"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type removeStubStorage struct {
	storage.IStorage

	removeErr map[string]error
	present   map[string]bool
	removed   []string
}

func (s *removeStubStorage) RemoveObject(path string) error {
	s.removed = append(s.removed, path)
	return s.removeErr[path]
}

func (s *removeStubStorage) StatObject(path string) (map[string]string, error) {
	if s.present[path] {
		return map[string]string{}, nil
	}
	return nil, errors.New("not found")
}

func TestRemoveVersionObjects(t *testing.T) {
	t.Run("one failing key does not stop the others", func(t *testing.T) {
		store := &removeStubStorage{
			removeErr: map[string]error{"b": errors.New("boom")},
			present:   map[string]bool{"b": true},
		}

		failed := RemoveVersionObjects(store, zap.NewNop(), []string{"a", "b", "c"})

		assert.Equal(t, []string{"a", "b", "c"}, store.removed,
			"every key must be attempted, not just up to the first failure")
		assert.Equal(t, []string{"b"}, failed)
	})

	t.Run("an already absent key counts as removed", func(t *testing.T) {
		// GCS and Azure report not-found on delete while S3 deletes idempotently. Treating that as
		// a failure would make retrying callers loop forever on an object that is already gone.
		store := &removeStubStorage{
			removeErr: map[string]error{"gone": errors.New("object does not exist")},
			present:   map[string]bool{},
		}

		failed := RemoveVersionObjects(store, zap.NewNop(), []string{"gone"})

		assert.Empty(t, failed)
	})

	t.Run("no keys is not an error", func(t *testing.T) {
		store := &removeStubStorage{}

		assert.Empty(t, RemoveVersionObjects(store, zap.NewNop(), nil))
		assert.Empty(t, store.removed)
	})
}
