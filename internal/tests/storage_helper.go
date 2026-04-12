package tests

import (
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewTestMinioClient(t *testing.T, endpoint string, secure bool) *minio.Client {
	t.Helper()
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("testkey", "testsecret", ""),
		Secure: secure,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("minio.New(%q): %v", endpoint, err)
	}
	return client
}

func KeysOf(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
