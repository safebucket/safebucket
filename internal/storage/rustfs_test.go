package storage_test

import (
	"strings"
	"testing"

	"github.com/safebucket/safebucket/internal/storage"
	"github.com/safebucket/safebucket/internal/tests"
)

func TestS3Storage_PresignedGetObject_UsesSigningClient(t *testing.T) {
	internal := tests.NewTestMinioClient(t, "internal.cluster.local:9000", false)
	signing := tests.NewTestMinioClient(t, "public.example.com:443", true)

	s := storage.NewS3StorageForTest("test-bucket", internal, signing)

	rawURL, err := s.PresignedGetObject("buckets/abc/file.txt")
	if err != nil {
		t.Fatalf("PresignedGetObject: %v", err)
	}

	if !strings.Contains(rawURL, "public.example.com") {
		t.Errorf("expected URL to contain external host, got %q", rawURL)
	}
	if strings.Contains(rawURL, "internal.cluster.local") {
		t.Errorf("URL leaks internal host: %q", rawURL)
	}
	if !strings.HasPrefix(rawURL, "https://") {
		t.Errorf("expected https scheme derived from signing client, got %q", rawURL)
	}
	if !strings.Contains(rawURL, "X-Amz-Signature=") {
		t.Errorf("expected SigV4 signature query param, got %q", rawURL)
	}
}

func TestS3Storage_PresignedPostPolicy_UsesSigningClient(t *testing.T) {
	internal := tests.NewTestMinioClient(t, "internal.cluster.local:9000", false)
	signing := tests.NewTestMinioClient(t, "public.example.com:443", true)

	s := storage.NewS3StorageForTest("test-bucket", internal, signing)

	rawURL, formData, err := s.PresignedPostPolicy("buckets/abc/file.txt", 1024, map[string]string{
		"bucket_id": "bucket-1",
		"file_id":   "file-1",
		"user_id":   "user-1",
	})
	if err != nil {
		t.Fatalf("PresignedPostPolicy: %v", err)
	}

	if !strings.Contains(rawURL, "public.example.com") {
		t.Errorf("expected action URL to target external host, got %q", rawURL)
	}
	if strings.Contains(rawURL, "internal.cluster.local") {
		t.Errorf("action URL leaks internal host: %q", rawURL)
	}
	if _, ok := formData["x-amz-signature"]; !ok {
		t.Errorf("expected x-amz-signature form field, got keys %v", tests.KeysOf(formData))
	}
	if _, ok := formData["policy"]; !ok {
		t.Errorf("expected policy form field, got keys %v", tests.KeysOf(formData))
	}
}

func TestNewSigningClient_SchemeDerivedFromExternalURL(t *testing.T) {
	cases := []struct {
		name             string
		externalEndpoint string
		wantScheme       string
	}{
		{name: "https", externalEndpoint: "https://public.example.com", wantScheme: "https://"},
		{name: "http", externalEndpoint: "http://public.example.com", wantScheme: "http://"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			signing, err := storage.NewSigningClientForTest(
				tc.externalEndpoint, "testkey", "testsecret", "us-east-1", false,
			)
			if err != nil {
				t.Fatalf("NewSigningClientForTest: %v", err)
			}

			s := storage.NewS3StorageForTest(
				"test-bucket",
				tests.NewTestMinioClient(t, "internal.cluster.local:9000", false),
				signing,
			)

			rawURL, err := s.PresignedGetObject("buckets/abc/file.txt")
			if err != nil {
				t.Fatalf("PresignedGetObject: %v", err)
			}
			if !strings.HasPrefix(rawURL, tc.wantScheme) {
				t.Errorf("expected scheme %q, got %q", tc.wantScheme, rawURL)
			}
			if !strings.Contains(rawURL, "public.example.com") {
				t.Errorf("expected external host in URL, got %q", rawURL)
			}
		})
	}
}
