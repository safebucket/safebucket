package activity

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/blevesearch/bleve/v2"
)

func newTestFilesystemClient(t *testing.T) *FilesystemClient {
	t.Helper()
	dir := t.TempDir()
	config := models.ActivityConfiguration{
		Type: "filesystem",
		Filesystem: &models.FilesystemActivityConfiguration{
			Directory: dir,
		},
	}
	client := NewFilesystemClient(config).(*FilesystemClient)
	t.Cleanup(func() { client.Close() })
	return client
}

func sendTestActivity(
	t *testing.T, client *FilesystemClient,
	action, objectType, userID, bucketID, message string, ts time.Time,
) {
	t.Helper()
	err := client.Send(models.Activity{
		Message: message,
		Filter: models.LogFilter{
			Fields: models.ActivityFields{
				Action:     action,
				ObjectType: objectType,
				UserID:     userID,
				Domain:     "example.com",
				BucketID:   bucketID,
			},
			Timestamp: strconv.FormatInt(ts.UnixNano(), 10),
		},
		Object: map[string]any{"name": "test-object"},
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestFilesystemSendAndSearch(t *testing.T) {
	client := newTestFilesystemClient(t)

	now := time.Now()
	sendTestActivity(
		t, client, "create", "bucket", "user-1", "bucket-1", "Created bucket", now,
	)

	results, err := client.Search(map[string][]string{
		"action": {"create"},
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r["action"] != "create" {
		t.Errorf("expected action=create, got %v", r["action"])
	}
	if r["object_type"] != "bucket" {
		t.Errorf("expected object_type=bucket, got %v", r["object_type"])
	}
	if r["user_id"] != "user-1" {
		t.Errorf("expected user_id=user-1, got %v", r["user_id"])
	}
	if r["message"] != "Created bucket" {
		t.Errorf("expected message='Created bucket', got %v", r["message"])
	}
	if r["domain"] != "example.com" {
		t.Errorf("expected domain=example.com, got %v", r["domain"])
	}

	tsStr, ok := r["timestamp"].(string)
	if !ok {
		t.Fatal("timestamp should be a string")
	}
	if _, parseErr := strconv.ParseInt(tsStr, 10, 64); parseErr != nil {
		t.Errorf("timestamp should be parseable as int64: %v", parseErr)
	}

	obj, ok := r["object"].(map[string]any)
	if !ok {
		t.Fatal("object should be a map")
	}
	if obj["name"] != "test-object" {
		t.Errorf("expected object.name=test-object, got %v", obj["name"])
	}
}

func TestFilesystemSearchWithORCriteria(t *testing.T) {
	client := newTestFilesystemClient(t)

	now := time.Now()
	sendTestActivity(
		t, client, "create", "bucket", "user-1", "bucket-1", "Created bucket", now,
	)
	sendTestActivity(
		t, client, "delete", "bucket", "user-1", "bucket-2", "Deleted bucket", now.Add(-time.Second),
	)
	sendTestActivity(
		t, client, "update", "file", "user-2", "bucket-1", "Updated file", now.Add(-2*time.Second),
	)

	results, err := client.Search(map[string][]string{
		"action": {"create", "delete"},
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	actions := map[string]bool{}
	for _, r := range results {
		actions[r["action"].(string)] = true
	}
	if !actions["create"] || !actions["delete"] {
		t.Errorf("expected create and delete actions, got %v", actions)
	}
}

func TestFilesystemCountByDay(t *testing.T) {
	client := newTestFilesystemClient(t)

	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	sendTestActivity(
		t, client, "create", "bucket", "user-1", "bucket-1", "Created bucket 1", today,
	)
	sendTestActivity(
		t, client, "create", "bucket", "user-1", "bucket-2", "Created bucket 2", today.Add(-time.Minute),
	)
	sendTestActivity(
		t, client, "delete", "bucket", "user-1", "bucket-3", "Deleted bucket", yesterday,
	)

	points, err := client.CountByDay(map[string][]string{}, 7)
	if err != nil {
		t.Fatalf("CountByDay failed: %v", err)
	}

	totalCount := int64(0)
	for _, p := range points {
		totalCount += p.Count
	}

	if totalCount != 3 {
		t.Errorf("expected total count of 3, got %d (points: %+v)", totalCount, points)
	}
}

func TestFilesystemSearchRespectsTimeWindow(t *testing.T) {
	client := newTestFilesystemClient(t)

	oldTime := time.Now().AddDate(0, 0, -60)
	sendTestActivity(
		t, client, "create", "bucket", "user-1", "bucket-old", "Old event", oldTime,
	)

	sendTestActivity(
		t, client, "create", "bucket", "user-1", "bucket-new", "New event", time.Now(),
	)

	results, err := client.Search(map[string][]string{
		"action": {"create"},
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result (only recent), got %d", len(results))
	}

	if results[0]["bucket_id"] != "bucket-new" {
		t.Errorf("expected bucket_id=bucket-new, got %v", results[0]["bucket_id"])
	}
}

func TestFilesystemMigrateIndex(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "activity.bleve")

	indexMapping := buildIndexMapping()
	index, err := bleve.New(dir, indexMapping)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}
	err = index.SetInternal(schemaVersionKey, []byte("0"))
	if err != nil {
		t.Fatalf("failed to set schema version: %v", err)
	}

	now := time.Now()
	type testDoc struct {
		Message    string    `json:"message"`
		Timestamp  time.Time `json:"timestamp"`
		Action     string    `json:"action"`
		ObjectType string    `json:"object_type"`
		UserID     string    `json:"user_id"`
		Domain     string    `json:"domain"`
		BucketID   string    `json:"bucket_id"`
		FileID     string    `json:"file_id"`
		Object     string    `json:"object"`
	}

	docs := []testDoc{
		{
			Message:    "Created bucket",
			Timestamp:  now,
			Action:     "create",
			ObjectType: "bucket",
			UserID:     "user-1",
			Domain:     "example.com",
			BucketID:   "bucket-1",
			Object:     `{"name":"obj1"}`,
		},
		{
			Message:    "Deleted file",
			Timestamp:  now.Add(-time.Second),
			Action:     "delete",
			ObjectType: "file",
			UserID:     "user-2",
			Domain:     "example.com",
			BucketID:   "bucket-2",
			FileID:     "file-1",
		},
	}
	for i, doc := range docs {
		err = index.Index(strconv.Itoa(i), doc)
		if err != nil {
			t.Fatalf("failed to index doc %d: %v", i, err)
		}
	}

	err = index.Close()
	if err != nil {
		t.Fatalf("failed to close index: %v", err)
	}

	config := models.ActivityConfiguration{
		Type: "filesystem",
		Filesystem: &models.FilesystemActivityConfiguration{
			Directory: dir,
		},
	}
	client := NewFilesystemClient(config).(*FilesystemClient)

	storedVersion, err := client.index.GetInternal(schemaVersionKey)
	if err != nil {
		t.Fatalf("failed to get schema version: %v", err)
	}
	if string(storedVersion) != schemaVersion {
		t.Errorf("expected schema version %s, got %s", schemaVersion, string(storedVersion))
	}

	results, err := client.Search(map[string][]string{})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results after migration, got %d", len(results))
	}

	found := map[string]bool{}
	for _, r := range results {
		found[r["action"].(string)] = true
	}
	if !found["create"] || !found["delete"] {
		t.Errorf("expected create and delete actions after migration, got %v", found)
	}
}
