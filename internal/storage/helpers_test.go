package storage

import (
	"testing"

	c "github.com/safebucket/safebucket/internal/configuration"

	"github.com/stretchr/testify/assert"
)

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

func TestPartCount(t *testing.T) {
	const mib = int64(1 << 20)

	assert.Equal(t, 1, PartCount(32*mib, 32*mib))
	assert.Equal(t, 2, PartCount(32*mib+1, 32*mib))
	assert.Equal(t, 10000, PartCount(32*mib*10000, 32*mib))
}

func TestExpectedPartSize(t *testing.T) {
	const mib = int64(1 << 20)

	assert.Equal(t, 32*mib, ExpectedPartSize(64*mib, 32*mib, 1, 2))
	assert.Equal(t, 32*mib, ExpectedPartSize(64*mib, 32*mib, 2, 2))

	size := 32*mib + 1
	assert.Equal(t, 32*mib, ExpectedPartSize(size, 32*mib, 1, 2))
	assert.Equal(t, int64(1), ExpectedPartSize(size, 32*mib, 2, 2))
}
