//go:build integration

package integration

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"
)

// Can be override via the MINIO_IMAGE env var
const defaultMinIOImage = "minio/minio:RELEASE.2024-01-16T16-07-38Z"

type MinIOInstance struct {
	Endpoint         string
	ExternalEndpoint string
	AccessKey        string
	SecretKey        string
	Bucket           string
}

func StartMinIO(t *testing.T) MinIOInstance {
	t.Helper()

	ctx := context.Background()

	image := os.Getenv("MINIO_IMAGE")
	if image == "" {
		image = defaultMinIOImage
	}

	container, err := tcminio.Run(ctx, image)
	require.NoError(t, err, "start minio container")

	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(container)
	})

	endpoint, err := container.ConnectionString(ctx)
	require.NoError(t, err, "minio connection string")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(container.Username, container.Password, ""),
		Secure: false,
	})
	require.NoError(t, err, "init minio client")

	bucket := "sb-" + randomHex(6)
	require.NoError(t,
		client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}),
		"create bucket %s", bucket,
	)

	external := (&url.URL{Scheme: "http", Host: endpoint}).String()

	return MinIOInstance{
		Endpoint:         endpoint,
		ExternalEndpoint: external,
		AccessKey:        container.Username,
		SecretKey:        container.Password,
		Bucket:           bucket,
	}
}
