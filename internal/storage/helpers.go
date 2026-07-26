package storage

import (
	c "github.com/safebucket/safebucket/internal/configuration"
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
	return partSize, PartCount(size, partSize)
}

func PartCount(size, partSize int64) int {
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
