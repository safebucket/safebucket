package reconcile

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/safebucket/safebucket/internal/core"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const objectPrefix = "buckets/"

func Run(cfg models.Configuration, opts Options) error {
	db := core.NewDatabase(cfg.Database)
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	store := core.NewStorage(cfg.Storage, 0)

	result := newReport(opts.prefix(), opts)

	if err = scanObjects(db, store, result, opts); err != nil {
		return err
	}
	printSummary(result, opts)

	if opts.Apply {
		if err = checkSafety(result, opts); err != nil {
			return err
		}
		deleteOrphans(store, result)
	}

	if err = writeReport(opts.Report, result); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Report written to %s\n", opts.Report)

	printResult(result, opts)
	return nil
}

func scanObjects(
	db *gorm.DB,
	store storage.IStorage,
	result *report,
	opts Options,
) error {
	resources, err := loadResourceIDs(db)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-opts.Grace)

	err = store.IterateObjects(result.Prefix, func(object storage.ObjectInfo) error {
		result.Total.Count++
		result.Total.Size += object.Size

		resourceID, ok := parseResourceID(object.Key)
		if !ok {
			result.Unknown.Count++
			result.Unknown.Size += object.Size
			result.UnknownKeys = append(result.UnknownKeys, object.Key)
			return nil
		}

		if _, referenced := resources[resourceID]; referenced {
			result.Referenced.Count++
			result.Referenced.Size += object.Size
			return nil
		}

		if object.LastModified.After(cutoff) {
			result.TooRecent.Count++
			result.TooRecent.Size += object.Size
			return nil
		}

		result.Orphaned.Count++
		result.Orphaned.Size += object.Size
		result.OrphanedObjects = append(result.OrphanedObjects, objectRecord{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
		})
		return nil
	})
	if err != nil {
		return fmt.Errorf("list storage objects: %w", err)
	}

	return nil
}

func checkSafety(result *report, opts Options) error {
	if !opts.Force && result.Orphaned.Count*2 >= result.Total.Count {
		return fmt.Errorf(
			"%d of %d scanned objects are orphaned. Re-run with --force to delete them",
			result.Orphaned.Count,
			result.Total.Count,
		)
	}

	return nil
}

func loadResourceIDs(db *gorm.DB) (map[uuid.UUID]struct{}, error) {
	var fileIDs, folderIDs []uuid.UUID

	if err := db.Unscoped().Model(&models.File{}).Pluck("id", &fileIDs).Error; err != nil {
		return nil, fmt.Errorf("load file IDs: %w", err)
	}
	if err := db.Unscoped().Model(&models.Folder{}).Pluck("id", &folderIDs).Error; err != nil {
		return nil, fmt.Errorf("load folder IDs: %w", err)
	}

	resources := make(map[uuid.UUID]struct{}, len(fileIDs)+len(folderIDs))
	for _, id := range append(fileIDs, folderIDs...) {
		resources[id] = struct{}{}
	}

	return resources, nil
}

func parseResourceID(key string) (uuid.UUID, bool) {
	remainder, found := strings.CutPrefix(key, objectPrefix)
	if !found {
		return uuid.Nil, false
	}

	bucketPart, resourcePart, found := strings.Cut(remainder, "/")
	if !found || strings.Contains(resourcePart, "/") {
		return uuid.Nil, false
	}

	if _, err := uuid.Parse(bucketPart); err != nil {
		return uuid.Nil, false
	}

	resourceID, err := uuid.Parse(resourcePart)
	if err != nil {
		return uuid.Nil, false
	}

	return resourceID, true
}

func deleteOrphans(store storage.IStorage, result *report) {
	result.Applied = true

	for _, object := range result.OrphanedObjects {
		if err := store.RemoveObject(object.Key); err != nil {
			result.Failed = append(result.Failed, deleteFailure{Key: object.Key, Error: err.Error()})
			continue
		}

		result.DeletedCount++
		result.DeletedSize += object.Size
	}
}
