package storage

import (
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

type signingClientOptions struct {
	externalEndpoint string
	accessKey        string
	secretKey        string
	region           string
	forcePathStyle   bool
}

// newSigningClient builds a *minio.Client targeted at the external storage endpoint.
func newSigningClient(opts signingClientOptions) (*minio.Client, error) {
	parsed, err := url.Parse(opts.externalEndpoint)
	if err != nil {
		zap.L().Error("Failed to parse external endpoint",
			zap.String("endpoint", opts.externalEndpoint), zap.Error(err))
		return nil, err
	}

	minioOpts := &minio.Options{
		Creds:  credentials.NewStaticV4(opts.accessKey, opts.secretKey, ""),
		Secure: parsed.Scheme == "https",
		Region: opts.region,
	}
	if opts.forcePathStyle {
		minioOpts.BucketLookup = minio.BucketLookupPath
	}

	return minio.New(parsed.Host, minioOpts)
}
