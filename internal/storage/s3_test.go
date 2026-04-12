package storage

import (
	"strings"
	"testing"
)

func TestGenericS3Storage_PresignedGetObject_UsesSigningClient(t *testing.T) {
	internal := newTestMinioClient(t, "internal.cluster.local:9000", false)
	signing := newTestMinioClient(t, "s3.public.example.com:443", true)

	s := &GenericS3Storage{
		BucketName:    "test-bucket",
		storage:       internal,
		signingClient: signing,
	}

	rawURL, err := s.PresignedGetObject("buckets/abc/file.txt")
	if err != nil {
		t.Fatalf("PresignedGetObject: %v", err)
	}

	if !strings.Contains(rawURL, "s3.public.example.com") {
		t.Errorf("expected URL to contain external host, got %q", rawURL)
	}
	if strings.Contains(rawURL, "internal.cluster.local") {
		t.Errorf("URL leaks internal host: %q", rawURL)
	}
	if !strings.Contains(rawURL, "X-Amz-Signature=") {
		t.Errorf("expected SigV4 signature query param, got %q", rawURL)
	}
}

func TestGenericS3Storage_PresignedPostPolicy_UsesSigningClient(t *testing.T) {
	internal := newTestMinioClient(t, "internal.cluster.local:9000", false)
	signing := newTestMinioClient(t, "s3.public.example.com:443", true)

	s := &GenericS3Storage{
		BucketName:    "test-bucket",
		storage:       internal,
		signingClient: signing,
	}

	rawURL, formData, err := s.PresignedPostPolicy("buckets/abc/file.txt", 1024, map[string]string{
		"bucket_id": "bucket-1",
		"file_id":   "file-1",
		"user_id":   "user-1",
	})
	if err != nil {
		t.Fatalf("PresignedPostPolicy: %v", err)
	}

	if !strings.Contains(rawURL, "s3.public.example.com") {
		t.Errorf("expected action URL to target external host, got %q", rawURL)
	}
	if strings.Contains(rawURL, "internal.cluster.local") {
		t.Errorf("action URL leaks internal host: %q", rawURL)
	}
	if _, ok := formData["x-amz-signature"]; !ok {
		t.Errorf("expected x-amz-signature form field, got keys %v", keysOf(formData))
	}
	if _, ok := formData["policy"]; !ok {
		t.Errorf("expected policy form field, got keys %v", keysOf(formData))
	}
}

func TestGenericS3Storage_PresignedURL_HonorsForcePathStyle(t *testing.T) {
	signing, err := newSigningClient(signingClientOptions{
		externalEndpoint: "https://s3.public.example.com",
		accessKey:        "testkey",
		secretKey:        "testsecret",
		region:           "us-east-1",
		forcePathStyle:   true,
	})
	if err != nil {
		t.Fatalf("newSigningClient: %v", err)
	}

	s := &GenericS3Storage{
		BucketName:    "test-bucket",
		storage:       newTestMinioClient(t, "internal.cluster.local:9000", false),
		signingClient: signing,
	}

	rawURL, err := s.PresignedGetObject("buckets/abc/file.txt")
	if err != nil {
		t.Fatalf("PresignedGetObject: %v", err)
	}

	if !strings.Contains(rawURL, "s3.public.example.com/test-bucket/") {
		t.Errorf("expected path-style URL, got %q", rawURL)
	}
	if strings.Contains(rawURL, "test-bucket.s3.public.example.com") {
		t.Errorf("URL is virtual-hosted, expected path-style: %q", rawURL)
	}
}

func TestGenericS3Storage_LiveOps_UseInternalClient(t *testing.T) {
	signing := newTestMinioClient(t, "s3.public.example.com:443", true)
	s := &GenericS3Storage{
		BucketName:    "test-bucket",
		storage:       nil,
		signingClient: signing,
	}

	if _, err := s.PresignedGetObject("buckets/abc/file.txt"); err != nil {
		t.Fatalf("presign should succeed via signing client: %v", err)
	}

	defer func() {
		if recover() == nil {
			t.Errorf("expected StatObject to panic with nil storage client")
		}
	}()
	_, _ = s.StatObject("buckets/abc/file.txt")
}
