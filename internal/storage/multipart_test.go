package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubStorage struct {
	listUploadedPartsFn func(path, uploadID string) ([]PartInfo, error)
	completeMultipartFn func(path, uploadID string, parts []PartInfo) error
}

func (s *stubStorage) ListUploadedParts(path, uploadID string) ([]PartInfo, error) {
	return s.listUploadedPartsFn(path, uploadID)
}

func (s *stubStorage) CompleteMultipartUpload(path, uploadID string, parts []PartInfo) error {
	if s.completeMultipartFn != nil {
		return s.completeMultipartFn(path, uploadID, parts)
	}
	return nil
}

func (s *stubStorage) PresignedGetObject(string, GetObjectOptions) (string, error) { return "", nil }
func (s *stubStorage) PresignUpload(string, int, map[string]string) (PresignedUpload, error) {
	return PresignedUpload{}, nil
}
func (s *stubStorage) SupportsMultipart() bool                      { return true }
func (s *stubStorage) AbortMultipartUpload(string, string) error    { return nil }
func (s *stubStorage) StatObject(string) (map[string]string, error) { return nil, nil }
func (s *stubStorage) ListObjects(string, int32) ([]string, error)  { return nil, nil }
func (s *stubStorage) RemoveObject(string) error                    { return nil }
func (s *stubStorage) RemoveObjects([]string) error                 { return nil }
func (s *stubStorage) EnsureTrashLifecyclePolicy(int) error         { return nil }
func (s *stubStorage) MarkAsTrashed(string, interface{}) error      { return nil }
func (s *stubStorage) UnmarkAsTrashed(string, interface{}) error    { return nil }
func (s *stubStorage) IsTrashMarkerPath(string) (bool, string)      { return false, "" }
func (s *stubStorage) GetBucketName() string                        { return "" }

func TestFinalizeMultipartUpload(t *testing.T) {
	const mib = int64(1 << 20)
	const testUploadID = "upload-1"
	const testPartSize = int64(1 << 25) // 32 MiB

	t.Run("missing part", func(t *testing.T) {
		store := &stubStorage{
			listUploadedPartsFn: func(string, string) ([]PartInfo, error) {
				return []PartInfo{{PartNumber: 1, Size: 32 * mib}}, nil
			},
		}

		err := FinalizeMultipartUpload(store, "path", testUploadID, testPartSize, 32*mib+1)
		assert.ErrorIs(t, err, ErrMultipartPartMismatch)
	})

	t.Run("wrong mid-part size", func(t *testing.T) {
		store := &stubStorage{
			listUploadedPartsFn: func(string, string) ([]PartInfo, error) {
				return []PartInfo{
					{PartNumber: 1, Size: 31 * mib},
					{PartNumber: 2, Size: 32 * mib},
				}, nil
			},
		}

		err := FinalizeMultipartUpload(store, "path", testUploadID, testPartSize, 64*mib)
		assert.ErrorIs(t, err, ErrMultipartPartMismatch)
	})

	t.Run("wrong last-part size", func(t *testing.T) {
		store := &stubStorage{
			listUploadedPartsFn: func(string, string) ([]PartInfo, error) {
				return []PartInfo{
					{PartNumber: 1, Size: 32 * mib},
					{PartNumber: 2, Size: 2}, // expected remainder is 1 byte
				}, nil
			},
		}

		err := FinalizeMultipartUpload(store, "path", testUploadID, testPartSize, 32*mib+1)
		assert.ErrorIs(t, err, ErrMultipartPartMismatch)
	})

	t.Run("list parts storage error", func(t *testing.T) {
		store := &stubStorage{
			listUploadedPartsFn: func(string, string) ([]PartInfo, error) {
				return nil, assert.AnError
			},
		}

		err := FinalizeMultipartUpload(store, "path", testUploadID, testPartSize, 32*mib)
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrMultipartPartMismatch)
	})

	t.Run("complete storage error", func(t *testing.T) {
		store := &stubStorage{
			listUploadedPartsFn: func(string, string) ([]PartInfo, error) {
				return []PartInfo{{PartNumber: 1, Size: 32 * mib}}, nil
			},
			completeMultipartFn: func(string, string, []PartInfo) error {
				return assert.AnError
			},
		}

		err := FinalizeMultipartUpload(store, "path", testUploadID, testPartSize, 32*mib)
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrMultipartPartMismatch)
	})

	t.Run("happy path", func(t *testing.T) {
		completed := false
		store := &stubStorage{
			listUploadedPartsFn: func(string, string) ([]PartInfo, error) {
				return []PartInfo{
					{PartNumber: 1, Size: 32 * mib},
					{PartNumber: 2, Size: 1},
				}, nil
			},
			completeMultipartFn: func(_, uploadID string, parts []PartInfo) error {
				completed = true
				assert.Equal(t, testUploadID, uploadID)
				assert.Len(t, parts, 2)
				return nil
			},
		}

		err := FinalizeMultipartUpload(store, "path", testUploadID, testPartSize, 32*mib+1)
		require.NoError(t, err)
		assert.True(t, completed)
	})
}
