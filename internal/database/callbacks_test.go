package database

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// testUUIDModel is a test model with a UUID primary key.
type testUUIDModel struct {
	ID   uuid.UUID `gorm:"type:text;primaryKey"`
	Name string
}

// testAutoIncrModel is a test model with an auto-increment integer primary key.
type testAutoIncrModel struct {
	ID   uint `gorm:"primaryKey;autoIncrement"`
	Name string
}

func setupCallbackTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	RegisterCallbacks(db)

	require.NoError(t, db.AutoMigrate(&testUUIDModel{}, &testAutoIncrModel{}))

	return db
}

func TestGenerateUUID_AssignsUUID(t *testing.T) {
	db := setupCallbackTestDB(t)

	m := testUUIDModel{Name: "test"}
	require.Equal(t, uuid.Nil, m.ID)

	require.NoError(t, db.Create(&m).Error)
	assert.NotEqual(t, uuid.Nil, m.ID)
}

func TestGenerateUUID_PreservesExistingUUID(t *testing.T) {
	db := setupCallbackTestDB(t)

	existing := uuid.New()
	m := testUUIDModel{ID: existing, Name: "preset"}

	require.NoError(t, db.Create(&m).Error)
	assert.Equal(t, existing, m.ID)
}

func TestGenerateUUID_SkipsNonUUIDPrimaryKey(t *testing.T) {
	db := setupCallbackTestDB(t)

	m := testAutoIncrModel{Name: "auto"}
	require.NoError(t, db.Create(&m).Error)
	assert.NotZero(t, m.ID, "auto-increment ID should be assigned by the database")
}

func TestGenerateUUID_HandlesSliceCreate(t *testing.T) {
	db := setupCallbackTestDB(t)

	models := []testUUIDModel{
		{Name: "first"},
		{Name: "second"},
		{Name: "third"},
	}

	require.NoError(t, db.Create(&models).Error)

	seen := make(map[uuid.UUID]bool)
	for _, m := range models {
		assert.NotEqual(t, uuid.Nil, m.ID)
		assert.False(t, seen[m.ID], "each model should have a unique UUID")
		seen[m.ID] = true
	}
}
