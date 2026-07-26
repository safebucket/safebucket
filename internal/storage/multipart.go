package storage

import (
	"errors"
	"fmt"
)

var (
	ErrMultipartNotSupported = errors.New("multipart upload not supported by storage provider")
	ErrMultipartPartMismatch = errors.New("multipart parts do not match declared size")
)

func FinalizeMultipartUpload(store IStorage, objectPath, uploadID string, partSize, fileSize int64) error {
	parts, err := store.ListUploadedParts(objectPath, uploadID)
	if err != nil {
		return fmt.Errorf("list uploaded parts: %w", err)
	}

	partCount := PartCount(fileSize, partSize)
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
