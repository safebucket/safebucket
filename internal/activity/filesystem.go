package activity

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"api/internal/models"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const schemaVersion = "1"

var schemaVersionKey = []byte("schema_version")

// FilesystemActivityEntry is the document shape indexed in bleve.
type FilesystemActivityEntry struct {
	Message           string    `json:"message"`
	Timestamp         time.Time `json:"timestamp"`
	Action            string    `json:"action"`
	ObjectType        string    `json:"object_type"`
	UserID            string    `json:"user_id"`
	Domain            string    `json:"domain"`
	BucketID          string    `json:"bucket_id"`
	FileID            string    `json:"file_id"`
	FolderID          string    `json:"folder_id"`
	BucketMemberEmail string    `json:"bucket_member_email"`
	Object            string    `json:"object"`
}

// FilesystemClient implements IActivityLogger using a local bleve index.
type FilesystemClient struct {
	index bleve.Index
}

// NewFilesystemClient creates a FilesystemClient backed by a bleve index at the configured directory.
// If an existing index has a different schema version, it is automatically migrated.
func NewFilesystemClient(config models.ActivityConfiguration) IActivityLogger {
	dir := config.Filesystem.Directory

	index, err := bleve.Open(dir)
	if err != nil {
		indexMapping := buildIndexMapping()
		index, err = bleve.New(dir, indexMapping)
		if err != nil {
			zap.L().Fatal("Failed to create filesystem activity index", zap.Error(err))
		}
		err = index.SetInternal(schemaVersionKey, []byte(schemaVersion))
		if err != nil {
			zap.L().Fatal("Failed to set schema version", zap.Error(err))
		}
		return &FilesystemClient{index: index}
	}

	storedVersion, err := index.GetInternal(schemaVersionKey)
	if err != nil {
		zap.L().Fatal("Failed to get schema version", zap.Error(err))
	}

	if string(storedVersion) != schemaVersion {
		zap.L().Info("Schema version mismatch, migrating index",
			zap.String("old_version", string(storedVersion)),
			zap.String("new_version", schemaVersion),
		)
		err = index.Close()
		if err != nil {
			zap.L().Fatal("Failed to close old index before migration", zap.Error(err))
		}
		err = migrateIndex(dir)
		if err != nil {
			zap.L().Fatal("Failed to migrate index", zap.Error(err))
		}
		index, err = bleve.Open(dir)
		if err != nil {
			zap.L().Fatal("Failed to open migrated index", zap.Error(err))
		}
		err = index.SetInternal(schemaVersionKey, []byte(schemaVersion))
		if err != nil {
			zap.L().Fatal("Failed to set schema version after migration", zap.Error(err))
		}
	}

	return &FilesystemClient{index: index}
}

func buildIndexMapping() *mapping.IndexMappingImpl {
	keywordMapping := bleve.NewKeywordFieldMapping()
	dateMapping := bleve.NewDateTimeFieldMapping()
	textMapping := bleve.NewTextFieldMapping()

	disabledMapping := bleve.NewTextFieldMapping()
	disabledMapping.Index = false
	disabledMapping.Store = true

	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("action", keywordMapping)
	docMapping.AddFieldMappingsAt("object_type", keywordMapping)
	docMapping.AddFieldMappingsAt("user_id", keywordMapping)
	docMapping.AddFieldMappingsAt("domain", keywordMapping)
	docMapping.AddFieldMappingsAt("bucket_id", keywordMapping)
	docMapping.AddFieldMappingsAt("file_id", keywordMapping)
	docMapping.AddFieldMappingsAt("folder_id", keywordMapping)
	docMapping.AddFieldMappingsAt("bucket_member_email", keywordMapping)
	docMapping.AddFieldMappingsAt("timestamp", dateMapping)
	docMapping.AddFieldMappingsAt("message", textMapping)
	docMapping.AddFieldMappingsAt("object", disabledMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = docMapping

	return indexMapping
}

func cleanupOnError(indexes []bleve.Index, dirs []string) {
	for _, idx := range indexes {
		_ = idx.Close()
	}
	for _, d := range dirs {
		_ = os.RemoveAll(d)
	}
}

func copyDocuments(oldIndex, newIndex bleve.Index) (int, error) {
	const pageSize = 100
	from := 0
	totalMigrated := 0

	for {
		req := bleve.NewSearchRequest(bleve.NewMatchAllQuery())
		req.Size = pageSize
		req.From = from
		req.Fields = []string{"*"}

		result, err := oldIndex.Search(req)
		if err != nil {
			return 0, fmt.Errorf("failed to search old index: %w", err)
		}

		if len(result.Hits) == 0 {
			break
		}

		batch := newIndex.NewBatch()
		for _, hit := range result.Hits {
			entry := reconstructEntry(hit.Fields)
			if batchErr := batch.Index(hit.ID, entry); batchErr != nil {
				return 0, fmt.Errorf("failed to index document %s: %w", hit.ID, batchErr)
			}
		}

		err = newIndex.Batch(batch)
		if err != nil {
			return 0, fmt.Errorf("failed to batch index: %w", err)
		}

		totalMigrated += len(result.Hits)
		from += pageSize

		if len(result.Hits) < pageSize {
			break
		}
	}

	return totalMigrated, nil
}

func migrateIndex(dir string) error {
	oldIndex, err := bleve.Open(dir)
	if err != nil {
		return fmt.Errorf("failed to open old index: %w", err)
	}

	newDir := dir + ".new"
	newIndex, err := bleve.New(newDir, buildIndexMapping())
	if err != nil {
		_ = oldIndex.Close()
		return fmt.Errorf("failed to create new index: %w", err)
	}

	totalMigrated, err := copyDocuments(oldIndex, newIndex)
	if err != nil {
		cleanupOnError([]bleve.Index{oldIndex, newIndex}, []string{newDir})
		return err
	}

	zap.L().Info("Migration: copied documents", zap.Int("count", totalMigrated))

	err = oldIndex.Close()
	if err != nil {
		_ = newIndex.Close()
		return fmt.Errorf("failed to close old index: %w", err)
	}
	err = newIndex.Close()
	if err != nil {
		return fmt.Errorf("failed to close new index: %w", err)
	}

	oldDir := dir + ".old"
	err = os.Rename(dir, oldDir)
	if err != nil {
		return fmt.Errorf("failed to rename old index dir: %w", err)
	}
	err = os.Rename(newDir, dir)
	if err != nil {
		return fmt.Errorf("failed to rename new index dir: %w", err)
	}
	err = os.RemoveAll(oldDir)
	if err != nil {
		zap.L().Warn("Failed to remove old index dir", zap.Error(err))
	}

	return nil
}

func reconstructEntry(fields map[string]any) FilesystemActivityEntry {
	var entry FilesystemActivityEntry
	b, err := json.Marshal(fields)
	if err != nil {
		return entry
	}
	if unmarshalErr := json.Unmarshal(b, &entry); unmarshalErr != nil {
		return entry
	}
	return entry
}

func parseTimestamp(fields map[string]any) time.Time {
	if s, ok := fields["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (c *FilesystemClient) Close() error {
	return c.index.Close()
}

func (c *FilesystemClient) Send(activity models.Activity) error {
	ts, err := strconv.ParseInt(activity.Filter.Timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse timestamp: %w", err)
	}
	timestamp := time.Unix(0, ts)

	var objectJSON string
	if activity.Object != nil && isAuthorizedObject(activity.Filter.Fields["object_type"]) {
		var b []byte
		b, err = json.Marshal(activity.Object)
		if err != nil {
			return fmt.Errorf("failed to marshal object: %w", err)
		}
		objectJSON = string(b)
	}

	entry := FilesystemActivityEntry{
		Message:           activity.Message,
		Timestamp:         timestamp,
		Action:            activity.Filter.Fields["action"],
		ObjectType:        activity.Filter.Fields["object_type"],
		UserID:            activity.Filter.Fields["user_id"],
		Domain:            activity.Filter.Fields["domain"],
		BucketID:          activity.Filter.Fields["bucket_id"],
		FileID:            activity.Filter.Fields["file_id"],
		FolderID:          activity.Filter.Fields["folder_id"],
		BucketMemberEmail: activity.Filter.Fields["bucket_member_email"],
		Object:            objectJSON,
	}

	docID := uuid.New().String()
	err = c.index.Index(docID, entry)
	if err != nil {
		return fmt.Errorf("failed to index activity: %w", err)
	}

	return nil
}

func (c *FilesystemClient) Search(searchCriteria map[string][]string) ([]map[string]any, error) {
	criteriaQuery := buildBleveQuery(searchCriteria)

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	dateQuery := bleve.NewDateRangeQuery(thirtyDaysAgo, now)
	dateQuery.SetField("timestamp")

	conjunction := bleve.NewConjunctionQuery(criteriaQuery, dateQuery)

	searchRequest := bleve.NewSearchRequest(conjunction)
	searchRequest.Size = 100
	searchRequest.SortBy([]string{"-timestamp"})
	searchRequest.Fields = []string{"*"}

	result, err := c.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search activity: %w", err)
	}

	var activities []map[string]any
	for _, hit := range result.Hits {
		domain, _ := hit.Fields["domain"].(string)
		userID, _ := hit.Fields["user_id"].(string)
		action, _ := hit.Fields["action"].(string)
		objectType, _ := hit.Fields["object_type"].(string)
		bucketID, _ := hit.Fields["bucket_id"].(string)
		fileID, _ := hit.Fields["file_id"].(string)
		folderID, _ := hit.Fields["folder_id"].(string)
		bucketMemberEmail, _ := hit.Fields["bucket_member_email"].(string)
		message, _ := hit.Fields["message"].(string)

		entry := map[string]any{
			"domain":              domain,
			"user_id":             userID,
			"action":              action,
			"object_type":         objectType,
			"bucket_id":           bucketID,
			"file_id":             fileID,
			"folder_id":           folderID,
			"bucket_member_email": bucketMemberEmail,
			"message":             message,
		}

		if t := parseTimestamp(hit.Fields); !t.IsZero() {
			entry["timestamp"] = strconv.FormatInt(t.UnixNano(), 10)
		}

		if objectStr, _ := hit.Fields["object"].(string); objectStr != "" {
			var objectMap map[string]any
			if json.Unmarshal([]byte(objectStr), &objectMap) == nil {
				entry["object"] = objectMap
			}
		}

		activities = append(activities, entry)
	}

	return activities, nil
}

func (c *FilesystemClient) CountByDay(searchCriteria map[string][]string, days int) ([]models.TimeSeriesPoint, error) {
	criteriaQuery := buildBleveQuery(searchCriteria)

	now := time.Now()
	startTime := now.AddDate(0, 0, -days)
	dateQuery := bleve.NewDateRangeQuery(startTime, now)
	dateQuery.SetField("timestamp")

	conjunction := bleve.NewConjunctionQuery(criteriaQuery, dateQuery)

	searchRequest := bleve.NewSearchRequest(conjunction)
	searchRequest.Size = 0

	facet := bleve.NewFacetRequest("timestamp", days+1)
	for i := days; i >= 0; i-- {
		dayStart := now.AddDate(0, 0, -i).Truncate(24 * time.Hour)
		dayEnd := dayStart.Add(24 * time.Hour)
		name := dayStart.Format("2006-01-02")
		facet.AddDateTimeRange(name, dayStart, dayEnd)
	}
	searchRequest.AddFacet("daily_counts", facet)

	result, err := c.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to count activity by day: %w", err)
	}

	dailyFacet, ok := result.Facets["daily_counts"]
	if !ok {
		return []models.TimeSeriesPoint{}, nil
	}

	points := make([]models.TimeSeriesPoint, 0, len(dailyFacet.DateRanges))
	for _, dr := range dailyFacet.DateRanges {
		if dr.Count > 0 {
			points = append(points, models.TimeSeriesPoint{
				Date:  dr.Name,
				Count: int64(dr.Count),
			})
		}
	}

	return points, nil
}

func buildBleveQuery(searchCriteria map[string][]string) query.Query {
	var queries []query.Query

	for key, values := range searchCriteria {
		if len(values) == 1 {
			termQuery := bleve.NewTermQuery(values[0])
			termQuery.SetField(key)
			queries = append(queries, termQuery)
		} else if len(values) > 1 {
			var termQueries []query.Query
			for _, v := range values {
				tq := bleve.NewTermQuery(v)
				tq.SetField(key)
				termQueries = append(termQueries, tq)
			}
			disjunction := bleve.NewDisjunctionQuery(termQueries...)
			disjunction.SetMin(1)
			queries = append(queries, disjunction)
		}
	}

	if len(queries) == 0 {
		return bleve.NewMatchAllQuery()
	}

	if len(queries) == 1 {
		return queries[0]
	}

	return bleve.NewConjunctionQuery(queries...)
}
