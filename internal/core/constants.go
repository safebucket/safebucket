package core

import "api/internal/configuration"

// RequiresUploadConfirmation returns true for storage providers without bucket notifications.
func RequiresUploadConfirmation(provider string) bool {
	return provider == configuration.ProviderS3
}
