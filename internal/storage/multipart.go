package storage

import (
	"errors"
	"fmt"

	c "github.com/safebucket/safebucket/internal/configuration"
)

var (
	ErrMultipartNotSupported = errors.New("multipart upload not supported by storage provider")
	ErrMultipartPartMismatch = errors.New("multipart parts do not match declared size")
)

const bytesPerMiB = 1 << 20

func ceilDiv(a, b int64) int64 {
	return (a + b - 1) / b
}

func ceilToMiB(n int64) int64 {
	return ceilDiv(n, bytesPerMiB) * bytesPerMiB
}

func ComputeMultipartLayout(size int64) (int64, int) {
	partSize := c.MultipartPartSize
	if minPartSize := ceilToMiB(ceilDiv(size, c.MultipartMaxParts)); minPartSize > partSize {
		partSize = minPartSize
	}
	return partSize, GetPartsCount(size, partSize)
}

func GetPartsCount(size, partSize int64) int {
	return int(ceilDiv(size, partSize))
}

func ExpectedPartSize(size, partSize int64, partNumber, partCount int) int64 {
	if partNumber != partCount {
		return partSize
	}
	if remainder := size % partSize; remainder != 0 {
		return remainder
	}
	return partSize
}


func FinalizeMultipartUpload(store IStorage, objectPath, uploadID string, partSize, fileSize int64) error {
	parts, err := store.ListObjectParts(objectPath, uploadID)
	if err != nil {
		return fmt.Errorf("list uploaded parts: %w", err)
	}

	partCount := GetPartsCount(fileSize, partSize)
	if len(parts) != partCount {
		return ErrMultipartPartMismatch
	}

	var total int64
	for _, part := range parts {
		if part.Size != ExpectedPartSize(fileSize, partSize, part.PartNumber, partCount) {
			return ErrMultipartPartMismatch
		}
		total += part.Size
	}
	if total != fileSize {
		return ErrMultipartPartMismatch
	}

	if err = store.CompleteMultipartUpload(objectPath, uploadID, parts); err != nil {
		return fmt.Errorf("complete multipart upload: %w", err)
	}

	return nil
}
