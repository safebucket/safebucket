package core

import "api/internal/configuration"

// RequiresUploadConfirmation returns true when the client must manually confirm uploads.
// This is needed for storage providers without bucket notifications (e.g. generic S3)
// or when the events provider is in-memory (no external broker receives storage events).
func RequiresUploadConfirmation(storageProvider, eventsProvider string) bool {
	return storageProvider == configuration.ProviderS3 || eventsProvider == configuration.ProviderMemory
}
