package services

import (
	"net/http"
	"testing"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubStorage struct {
	listUploadedPartsFn func(path, uploadID string) ([]storage.PartInfo, error)
	completeMultipartFn func(path, uploadID string, parts []storage.PartInfo) error
}

func (s *stubStorage) PresignedGetObject(string, storage.GetObjectOptions) (string, error) {
	return "", nil
}

func (s *stubStorage) PresignUpload(string, int, map[string]string) (storage.PresignedUpload, error) {
	return storage.PresignedUpload{}, nil
}

func (s *stubStorage) SupportsMultipart() bool { return true }

func (s *stubStorage) ListUploadedParts(path, uploadID string) ([]storage.PartInfo, error) {
	return s.listUploadedPartsFn(path, uploadID)
}

func (s *stubStorage) CompleteMultipartUpload(path, uploadID string, parts []storage.PartInfo) error {
	if s.completeMultipartFn != nil {
		return s.completeMultipartFn(path, uploadID, parts)
	}
	return nil
}

func (s *stubStorage) AbortMultipartUpload(string, string) error { return nil }

func (s *stubStorage) StatObject(string) (map[string]string, error) { return nil, nil }
func (s *stubStorage) ListObjects(string, int32) ([]string, error)  { return nil, nil }
func (s *stubStorage) RemoveObject(string) error                    { return nil }
func (s *stubStorage) RemoveObjects([]string) error                 { return nil }
func (s *stubStorage) EnsureTrashLifecyclePolicy(int) error         { return nil }
func (s *stubStorage) MarkAsTrashed(string, any) error              { return nil }
func (s *stubStorage) UnmarkAsTrashed(string, any) error            { return nil }
func (s *stubStorage) IsTrashMarkerPath(string) (bool, string)      { return false, "" }
func (s *stubStorage) GetBucketName() string                        { return "" }

func TestCompleteMultipartUpload(t *testing.T) {
	const mib = int64(1 << 20)
	const testUploadID = "upload-1"
	const testPartSize = int64(1 << 25) // 32 MiB
	logger := zap.NewNop()

	t.Run("missing part", func(t *testing.T) {
		store := &stubStorage{
			listUploadedPartsFn: func(string, string) ([]storage.PartInfo, error) {
				return []storage.PartInfo{{PartNumber: 1, Size: 32 * mib}}, nil
			},
		}

		err := completeMultipartUpload(logger, store, "path", testUploadID, testPartSize, int(32*mib+1))
		requireAPIError(t, err, http.StatusBadRequest, apierrors.CodeMultipartSizeMismatch)
	})

	t.Run("wrong mid-part size", func(t *testing.T) {
		store := &stubStorage{
			listUploadedPartsFn: func(string, string) ([]storage.PartInfo, error) {
				return []storage.PartInfo{
					{PartNumber: 1, Size: 31 * mib},
					{PartNumber: 2, Size: 32 * mib},
				}, nil
			},
		}

		err := completeMultipartUpload(logger, store, "path", testUploadID, testPartSize, int(64*mib))
		requireAPIError(t, err, http.StatusBadRequest, apierrors.CodeMultipartSizeMismatch)
	})

	t.Run("wrong last-part size", func(t *testing.T) {
		store := &stubStorage{
			listUploadedPartsFn: func(string, string) ([]storage.PartInfo, error) {
				return []storage.PartInfo{
					{PartNumber: 1, Size: 32 * mib},
					{PartNumber: 2, Size: 2}, // expected remainder is 1 byte
				}, nil
			},
		}

		err := completeMultipartUpload(logger, store, "path", testUploadID, testPartSize, int(32*mib+1))
		requireAPIError(t, err, http.StatusBadRequest, apierrors.CodeMultipartSizeMismatch)
	})

	t.Run("list parts storage error", func(t *testing.T) {
		store := &stubStorage{
			listUploadedPartsFn: func(string, string) ([]storage.PartInfo, error) {
				return nil, assert.AnError
			},
		}

		err := completeMultipartUpload(logger, store, "path", testUploadID, testPartSize, int(32*mib))
		requireAPIError(t, err, http.StatusInternalServerError, apierrors.CodeMultipartCompleteFailed)
	})

	t.Run("complete storage error", func(t *testing.T) {
		store := &stubStorage{
			listUploadedPartsFn: func(string, string) ([]storage.PartInfo, error) {
				return []storage.PartInfo{{PartNumber: 1, Size: 32 * mib}}, nil
			},
			completeMultipartFn: func(string, string, []storage.PartInfo) error {
				return assert.AnError
			},
		}

		err := completeMultipartUpload(logger, store, "path", testUploadID, testPartSize, int(32*mib))
		requireAPIError(t, err, http.StatusInternalServerError, apierrors.CodeMultipartCompleteFailed)
	})

	t.Run("happy path", func(t *testing.T) {
		completed := false
		store := &stubStorage{
			listUploadedPartsFn: func(string, string) ([]storage.PartInfo, error) {
				return []storage.PartInfo{
					{PartNumber: 1, Size: 32 * mib},
					{PartNumber: 2, Size: 1},
				}, nil
			},
			completeMultipartFn: func(_, uploadID string, parts []storage.PartInfo) error {
				completed = true
				assert.Equal(t, testUploadID, uploadID)
				assert.Len(t, parts, 2)
				return nil
			},
		}

		err := completeMultipartUpload(logger, store, "path", testUploadID, testPartSize, int(32*mib+1))
		require.NoError(t, err)
		assert.True(t, completed)
	})
}
