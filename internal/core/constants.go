package core

import "github.com/safebucket/safebucket/internal/configuration"

func RequiresUploadConfirmation(storageProvider, eventsProvider string) bool {
	return storageProvider == configuration.ProviderS3 || eventsProvider == configuration.ProviderMemory
}
