package eventparser

import (
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/storage"
)

func NewBucketEventParser(storageType string, store storage.IStorage) IBucketEventParser {
	switch storageType {
	case configuration.ProviderRustFS:
		return &RustFSEventParser{}
	case configuration.ProviderMinio:
		return &MinIOEventParser{}
	case configuration.ProviderAWS:
		return &AWSEventParser{Storage: store}
	case configuration.ProviderGCP:
		return &GCPEventParser{}
	case configuration.ProviderAzure:
		return &AzureEventParser{Storage: store}
	case configuration.ProviderS3:
		return &MinIOEventParser{}
	default:
		return &RustFSEventParser{}
	}
}
