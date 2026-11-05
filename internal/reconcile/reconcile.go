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

	result, err := scanObjects(db, store, opts)
	if err != nil {
		return err
	}
	printSummary(result, opts)

	if !opts.Apply {
		fmt.Fprintf(os.Stdout, "Nothing was deleted. Re-run with --apply to delete %d orphaned objects (%s).\n",
			result.Orphaned.Count, formatSize(result.Orphaned.Size))
		return nil
	}

	deleteOrphans(store, result.Orphans)
	return nil
}

func scanObjects(db *gorm.DB, store storage.IStorage, opts Options) (*scanResult, error) {
	cutoff := time.Now().Add(-opts.Grace)

	resources, err := loadResourceIDs(db)
	if err != nil {
		return nil, err
	}

	result := &scanResult{}
	err = store.IterateObjects(objectPrefix, func(object storage.ObjectInfo) error {
		result.Total.add(object.Size)

		resourceID, ok := parseResourceID(object.Key)
		switch {
		case !ok:
			result.Unknown.add(object.Size)
		case resources[resourceID]:
			result.Referenced.add(object.Size)
		case object.LastModified.After(cutoff):
			result.TooRecent.add(object.Size)
		default:
			result.Orphaned.add(object.Size)
			result.Orphans = append(result.Orphans, object)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list storage objects: %w", err)
	}

	return result, nil
}

func loadResourceIDs(db *gorm.DB) (map[uuid.UUID]bool, error) {
	var fileIDs, folderIDs []uuid.UUID

	if err := db.Unscoped().Model(&models.File{}).Pluck("id", &fileIDs).Error; err != nil {
		return nil, fmt.Errorf("load file IDs: %w", err)
	}
	if err := db.Unscoped().Model(&models.Folder{}).Pluck("id", &folderIDs).Error; err != nil {
		return nil, fmt.Errorf("load folder IDs: %w", err)
	}

	resources := make(map[uuid.UUID]bool, len(fileIDs)+len(folderIDs))
	for _, id := range append(fileIDs, folderIDs...) {
		resources[id] = true
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

func deleteOrphans(store storage.IStorage, orphans []storage.ObjectInfo) {
	var deleted stats
	failed := 0

	for _, object := range orphans {
		if err := store.RemoveObject(object.Key); err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "Failed to delete %s: %v\n", object.Key, err)
			continue
		}
		deleted.add(object.Size)
	}

	fmt.Fprintf(os.Stdout, "Deleted %d orphaned objects (%s), %d failed.\n",
		deleted.Count, formatSize(deleted.Size), failed)
}
