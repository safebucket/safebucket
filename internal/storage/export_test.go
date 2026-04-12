package storage

import "github.com/minio/minio-go/v7"

// NewSigningClientForTest wraps newSigningClient for external test packages.
func NewSigningClientForTest(
	externalEndpoint, accessKey, secretKey, region string,
	forcePathStyle bool,
) (*minio.Client, error) {
	return newSigningClient(signingClientOptions{
		externalEndpoint: externalEndpoint,
		accessKey:        accessKey,
		secretKey:        secretKey,
		region:           region,
		forcePathStyle:   forcePathStyle,
	})
}

// NewS3StorageForTest constructs an S3Storage with the given clients.
func NewS3StorageForTest(bucketName string, storageClient, signing *minio.Client) *S3Storage {
	return &S3Storage{BucketName: bucketName, storage: storageClient, signingClient: signing}
}

// NewGenericS3StorageForTest constructs a GenericS3Storage with the given clients.
func NewGenericS3StorageForTest(bucketName string, storageClient, signing *minio.Client) *GenericS3Storage {
	return &GenericS3Storage{BucketName: bucketName, storage: storageClient, signingClient: signing}
}
