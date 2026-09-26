//go:build integration

package migration_test

import (
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/database"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadVersionMigration(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			dialect := bootstrap.LoadScenario(t, scenario).Database.Type
			db := bootstrap.NewDBProvider(t, dialect).Connect(t)
			connection, err := db.DB()
			require.NoError(t, err)
			require.NoError(t, bootstrap.MigrateUpTo(connection, dialect, 14))
			database.RegisterCallbacks(db)
			seedDB(t, db)
			var bucket models.Bucket
			require.NoError(t, db.Find(&bucket).Error)
			statuses := []models.FileStatus{
				models.FileStatusUploading,
				models.FileStatusDeleted,
				models.FileStatusDeleting,
				models.FileStatusRestoring,
			}
			for _, status := range statuses {
				id := uuid.New()
				require.NoError(t, db.Table("files").Create(map[string]any{
					"id": id, "name": string(status), "bucket_id": bucket.ID, "status": status, "size": 12,
					"created_at": time.Now(), "updated_at": time.Now(),
				}).Error)
			}
			require.NoError(t, bootstrap.MigrateUpTo(connection, dialect, 15))
			var files []models.File
			require.NoError(t, db.Unscoped().Find(&files).Error)
			require.Len(t, files, len(statuses)+1)
			for _, file := range files {
				var version models.FileVersion
				require.NoError(t, db.Where("file_id = ?", file.ID).Find(&version).Error)
				assert.Equal(t, file.ID, version.ID)
				assert.Equal(t, 1, version.Version)
				assert.Equal(t, file.Size, version.Size)
				if file.Status == models.FileStatusUploading {
					assert.Nil(t, file.CurrentVersionID)
					assert.Equal(t, models.FileStatusUploading, version.Status)
				} else {
					assert.Equal(t, &file.ID, file.CurrentVersionID)
					assert.Equal(t, models.FileStatusUploaded, version.Status)
				}
			}
		})
	}
}
