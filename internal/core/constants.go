package core

const (
	ProviderJetstream = "jetstream"
	ProviderMinio     = "minio"
	ProviderGCP       = "gcp"
	ProviderAWS       = "aws"
	ProviderRustFS    = "rustfs"
	ProviderS3        = "s3"
)

// RequiresUploadConfirmation returns true for storage providers without bucket notifications.
func RequiresUploadConfirmation(provider string) bool {
	return provider == ProviderS3
}
