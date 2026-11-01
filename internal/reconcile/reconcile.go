package reconcile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/safebucket/safebucket/internal/core"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const objectPrefix = "buckets/"

type objectKey struct {
	bucketID   uuid.UUID
	resourceID uuid.UUID
}

type scanState struct {
	buckets        map[uuid.UUID]struct{}
	resources      map[uuid.UUID]struct{}
	missingBuckets map[uuid.UUID]struct{}
	cutoff         time.Time
	result         *report
}

func newScanState(
	result *report,
	buckets map[uuid.UUID]struct{},
	resources map[uuid.UUID]struct{},
	cutoff time.Time,
) *scanState {
	return &scanState{
		buckets:        buckets,
		resources:      resources,
		missingBuckets: make(map[uuid.UUID]struct{}),
		cutoff:         cutoff,
		result:         result,
	}
}

func (s *scanState) classify(object storage.ObjectInfo) {
	s.result.Total.Count++
	s.result.Total.Size += object.Size

	key, ok := parseObjectKey(object.Key)
	if !ok {
		s.result.Unknown.Count++
		s.result.Unknown.Size += object.Size
		s.result.UnknownKeys = append(s.result.UnknownKeys, object.Key)
		return
	}

	if _, referenced := s.resources[key.resourceID]; referenced {
		s.result.Referenced.Count++
		s.result.Referenced.Size += object.Size
		return
	}

	if object.LastModified.After(s.cutoff) {
		s.result.TooRecent.Count++
		s.result.TooRecent.Size += object.Size
		return
	}

	s.result.Orphaned.Count++
	s.result.Orphaned.Size += object.Size
	s.result.OrphanedObjects = append(s.result.OrphanedObjects, objectRecord{
		Key:          object.Key,
		Size:         object.Size,
		LastModified: object.LastModified,
	})

	if _, exists := s.buckets[key.bucketID]; !exists {
		s.missingBuckets[key.bucketID] = struct{}{}
	}
}

func (s *scanState) finalize() {
	for bucketID := range s.missingBuckets {
		s.result.MissingBucketIDs = append(s.result.MissingBucketIDs, bucketID.String())
	}
	sort.Strings(s.result.MissingBucketIDs)
}

func Run(ctx context.Context, cfg models.Configuration, opts Options) error {
	db := core.NewDatabase(cfg.Database)
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	store := core.NewStorage(cfg.Storage, 0)
	if store == nil {
		return fmt.Errorf("unsupported storage type %q", cfg.Storage.Type)
	}

	result := newReport(opts.prefix(), opts)

	bucketCount, scanErr := scanObjects(ctx, db, store, result, opts)
	safetyErr := checkSafety(result, opts, bucketCount)

	var applyErr error
	if scanErr == nil && safetyErr == nil && opts.Apply {
		applyErr = deleteOrphans(ctx, db, store, result)
	}

	writeErr := writeReport(opts.Report, result)

	if scanErr == nil {
		printSummary(result, opts)
	}
	if writeErr == nil {
		fmt.Fprintf(os.Stdout, "Report written to %s\n", opts.Report)
	}

	if err = errors.Join(scanErr, safetyErr, applyErr, writeErr); err != nil {
		return err
	}

	printResult(result, opts)
	return nil
}

func scanObjects(
	ctx context.Context,
	db *gorm.DB,
	store storage.IStorage,
	result *report,
	opts Options,
) (int, error) {
	buckets, resources, err := loadReferenceIDs(ctx, db)
	if err != nil {
		return 0, err
	}

	state := newScanState(result, buckets, resources, time.Now().Add(-opts.Grace))

	iterateErr := store.IterateObjects(result.Prefix, func(object storage.ObjectInfo) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		state.classify(object)
		return nil
	})
	state.finalize()

	if iterateErr != nil {
		return len(buckets), fmt.Errorf("list storage objects: %w", iterateErr)
	}

	return len(buckets), nil
}

func checkSafety(result *report, opts Options, bucketCount int) error {
	if bucketCount == 0 && result.Total.Count > 0 {
		return fmt.Errorf(
			"database has no buckets but storage contains %d objects; check the configured database",
			result.Total.Count,
		)
	}

	if opts.Apply && !opts.Force && result.Orphaned.Count*2 > result.Total.Count {
		return fmt.Errorf(
			"%d of %d scanned objects are orphaned, more than half; re-run with --force to delete them",
			result.Orphaned.Count,
			result.Total.Count,
		)
	}

	return nil
}

func loadReferenceIDs(
	ctx context.Context,
	db *gorm.DB,
) (map[uuid.UUID]struct{}, map[uuid.UUID]struct{}, error) {
	bucketIDs, err := pluckIDs(ctx, db, &models.Bucket{})
	if err != nil {
		return nil, nil, fmt.Errorf("load bucket IDs: %w", err)
	}

	fileIDs, err := pluckIDs(ctx, db, &models.File{})
	if err != nil {
		return nil, nil, fmt.Errorf("load file IDs: %w", err)
	}

	folderIDs, err := pluckIDs(ctx, db, &models.Folder{})
	if err != nil {
		return nil, nil, fmt.Errorf("load folder IDs: %w", err)
	}

	resources := make(map[uuid.UUID]struct{}, len(fileIDs)+len(folderIDs))
	for _, id := range fileIDs {
		resources[id] = struct{}{}
	}
	for _, id := range folderIDs {
		resources[id] = struct{}{}
	}

	return toSet(bucketIDs), resources, nil
}

func pluckIDs(ctx context.Context, db *gorm.DB, model any) ([]uuid.UUID, error) {
	rows, err := db.WithContext(ctx).Unscoped().Model(model).Select("id").Rows()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, scanErr
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func toSet(ids []uuid.UUID) map[uuid.UUID]struct{} {
	set := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

func parseObjectKey(key string) (objectKey, bool) {
	remainder, found := strings.CutPrefix(key, objectPrefix)
	if !found {
		return objectKey{}, false
	}

	bucketPart, resourcePart, found := strings.Cut(remainder, "/")
	if !found || strings.Contains(resourcePart, "/") {
		return objectKey{}, false
	}

	bucketID, err := uuid.Parse(bucketPart)
	if err != nil {
		return objectKey{}, false
	}

	resourceID, err := uuid.Parse(resourcePart)
	if err != nil {
		return objectKey{}, false
	}

	return objectKey{bucketID: bucketID, resourceID: resourceID}, true
}

func deleteOrphans(ctx context.Context, db *gorm.DB, store storage.IStorage, result *report) error {
	result.Applied = true

	for _, object := range result.OrphanedObjects {
		if err := ctx.Err(); err != nil {
			return err
		}

		key, ok := parseObjectKey(object.Key)
		if !ok {
			continue
		}

		exists, err := resourceExists(ctx, db, key.resourceID)
		if err != nil {
			result.Failed = append(result.Failed, deleteFailure{Key: object.Key, Error: err.Error()})
			continue
		}
		if exists {
			result.Skipped = append(result.Skipped, skippedObject{
				Key:    object.Key,
				Reason: "database row exists",
			})
			continue
		}

		if err = store.RemoveObject(object.Key); err != nil {
			result.Failed = append(result.Failed, deleteFailure{Key: object.Key, Error: err.Error()})
			continue
		}

		result.DeletedCount++
		result.DeletedSize += object.Size
	}

	return nil
}

func resourceExists(ctx context.Context, db *gorm.DB, id uuid.UUID) (bool, error) {
	for _, model := range []any{&models.File{}, &models.Folder{}} {
		var count int64
		if err := db.WithContext(ctx).Unscoped().Model(model).Where("id = ?", id).Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}

	return false, nil
}
