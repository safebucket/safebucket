package storage

import (
	"testing"

	c "github.com/safebucket/safebucket/internal/configuration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubStorage struct {
	listObjectPartsFn   func(path, uploadID string) ([]PartInfo, error)
	completeMultipartFn func(path, uploadID string, parts []PartInfo) error
}

func (s *stubStorage) ListObjectParts(path, uploadID string) ([]PartInfo, error) {
	return s.listObjectPartsFn(path, uploadID)
}

func (s *stubStorage) CompleteMultipartUpload(path, uploadID string, parts []PartInfo, _ map[string]string) error {
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
			listObjectPartsFn: func(string, string) ([]PartInfo, error) {
				return []PartInfo{{PartNumber: 1, Size: 32 * mib}}, nil
			},
		}

		err := FinalizeMultipartUpload(store, "path", testUploadID, testPartSize, 32*mib+1, nil)
		assert.ErrorIs(t, err, ErrMultipartPartMismatch)
	})

	t.Run("wrong mid-part size", func(t *testing.T) {
		store := &stubStorage{
			listObjectPartsFn: func(string, string) ([]PartInfo, error) {
				return []PartInfo{
					{PartNumber: 1, Size: 31 * mib},
					{PartNumber: 2, Size: 32 * mib},
				}, nil
			},
		}

		err := FinalizeMultipartUpload(store, "path", testUploadID, testPartSize, 64*mib, nil)
		assert.ErrorIs(t, err, ErrMultipartPartMismatch)
	})

	t.Run("wrong last-part size", func(t *testing.T) {
		store := &stubStorage{
			listObjectPartsFn: func(string, string) ([]PartInfo, error) {
				return []PartInfo{
					{PartNumber: 1, Size: 32 * mib},
					{PartNumber: 2, Size: 2}, // expected remainder is 1 byte
				}, nil
			},
		}

		err := FinalizeMultipartUpload(store, "path", testUploadID, testPartSize, 32*mib+1, nil)
		assert.ErrorIs(t, err, ErrMultipartPartMismatch)
	})

	t.Run("list parts storage error", func(t *testing.T) {
		store := &stubStorage{
			listObjectPartsFn: func(string, string) ([]PartInfo, error) {
				return nil, assert.AnError
			},
		}

		err := FinalizeMultipartUpload(store, "path", testUploadID, testPartSize, 32*mib, nil)
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrMultipartPartMismatch)
	})

	t.Run("complete storage error", func(t *testing.T) {
		store := &stubStorage{
			listObjectPartsFn: func(string, string) ([]PartInfo, error) {
				return []PartInfo{{PartNumber: 1, Size: 32 * mib}}, nil
			},
			completeMultipartFn: func(string, string, []PartInfo) error {
				return assert.AnError
			},
		}

		err := FinalizeMultipartUpload(store, "path", testUploadID, testPartSize, 32*mib, nil)
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrMultipartPartMismatch)
	})

	t.Run("happy path", func(t *testing.T) {
		completed := false
		store := &stubStorage{
			listObjectPartsFn: func(string, string) ([]PartInfo, error) {
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

		err := FinalizeMultipartUpload(store, "path", testUploadID, testPartSize, 32*mib+1, nil)
		require.NoError(t, err)
		assert.True(t, completed)
	})
}

func TestComputeMultipartLayout(t *testing.T) {
	const mib = int64(1 << 20)

	tests := []struct {
		name              string
		size              int64
		expectedPartSize  int64
		expectedPartCount int
	}{
		{
			name:              "exactly 64 MiB stays at the default part size",
			size:              64 * mib,
			expectedPartSize:  64 * mib,
			expectedPartCount: 1,
		},
		{
			name:              "64 MiB + 1 byte needs two parts",
			size:              64*mib + 1,
			expectedPartSize:  64 * mib,
			expectedPartCount: 2,
		},
		{
			name:              "exact 10000-part fit stays at the default part size",
			size:              64 * mib * c.MultipartMaxParts,
			expectedPartSize:  64 * mib,
			expectedPartCount: c.MultipartMaxParts,
		},
		{
			name:              "past the 10000-part fit scales the part size up",
			size:              64*mib*c.MultipartMaxParts + 1,
			expectedPartSize:  65 * mib,
			expectedPartCount: 9847,
		},
		{
			name:              "50 GiB default max upload size stays well under 10000 parts",
			size:              50 * 1024 * mib,
			expectedPartSize:  64 * mib,
			expectedPartCount: 800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partSize, partCount := ComputeMultipartLayout(tt.size)
			assert.Equal(t, tt.expectedPartSize, partSize)
			assert.Equal(t, tt.expectedPartCount, partCount)
		})
	}
}

func TestGetPartsCount(t *testing.T) {
	const mib = int64(1 << 20)

	assert.Equal(t, 1, GetPartsCount(32*mib, 32*mib))
	assert.Equal(t, 2, GetPartsCount(32*mib+1, 32*mib))
	assert.Equal(t, 10000, GetPartsCount(32*mib*10000, 32*mib))
}

func TestExpectedPartSize(t *testing.T) {
	const mib = int64(1 << 20)

	assert.Equal(t, 32*mib, ExpectedPartSize(64*mib, 32*mib, 1, 2))
	assert.Equal(t, 32*mib, ExpectedPartSize(64*mib, 32*mib, 2, 2))

	size := 32*mib + 1
	assert.Equal(t, 32*mib, ExpectedPartSize(size, 32*mib, 1, 2))
	assert.Equal(t, int64(1), ExpectedPartSize(size, 32*mib, 2, 2))
}
