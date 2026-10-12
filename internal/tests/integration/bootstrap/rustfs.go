//go:build integration

package bootstrap

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Can be override via the RUSTFS_IMAGE env var
const (
	defaultRustFSImage = "rustfs/rustfs:1.0.1"
	rustfsPort         = "9000/tcp"
	rustfsAccessKey    = "rustfsadmin"
	rustfsSecretKey    = "rustfsadmin"
)

type RustFSInstance struct {
	Endpoint         string
	ExternalEndpoint string
	AccessKey        string
	SecretKey        string
	Bucket           string
}

func StartRustFS(t *testing.T) RustFSInstance {
	t.Helper()

	ctx := context.Background()

	image := os.Getenv("RUSTFS_IMAGE")
	if image == "" {
		image = defaultRustFSImage
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			ExposedPorts: []string{rustfsPort},
			Env: map[string]string{
				"RUSTFS_ACCESS_KEY": rustfsAccessKey,
				"RUSTFS_SECRET_KEY": rustfsSecretKey,
			},
			WaitingFor: wait.ForHTTP("/health").
				WithPort(rustfsPort).
				WithStatusCodeMatcher(func(s int) bool { return s == 200 }),
		},
		Started: true,
	})
	require.NoError(t, err, "start rustfs container")

	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(container)
	})

	endpoint, err := container.PortEndpoint(ctx, rustfsPort, "")
	require.NoError(t, err, "rustfs endpoint")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(rustfsAccessKey, rustfsSecretKey, ""),
		Secure: false,
	})
	require.NoError(t, err, "init rustfs client")

	bucket := "sb-" + strings.Split(uuid.NewString(), "-")[0]
	require.NoError(t,
		client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}),
		"create bucket %s", bucket,
	)

	external := (&url.URL{Scheme: "http", Host: endpoint}).String()

	return RustFSInstance{
		Endpoint:         endpoint,
		ExternalEndpoint: external,
		AccessKey:        rustfsAccessKey,
		SecretKey:        rustfsSecretKey,
		Bucket:           bucket,
	}
}
