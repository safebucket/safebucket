package rbac

import (
	"database/sql"
	"testing"

	"api/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupMockDB creates a mock database for testing
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock, db
}

// TestGetUserMembership tests retrieving a user's membership for a bucket
func TestGetUserMembership(t *testing.T) {
	t.Run("should return membership when it exists", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()
		membershipID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group", "created_at", "updated_at", "deleted_at"}).
			AddRow(membershipID, userID, bucketID, "owner", nil, nil, nil)

		// GORM adds soft delete check, ORDER BY, and LIMIT for First() queries
		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(userID, bucketID, 1).
			WillReturnRows(rows)

		membership, err := GetUserMembership(gormDB, userID, bucketID)

		require.NoError(t, err)
		assert.NotNil(t, membership)
		assert.Equal(t, userID, membership.UserID)
		assert.Equal(t, bucketID, membership.BucketID)
		assert.Equal(t, models.GroupOwner, membership.Group)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should return nil when membership does not exist", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(userID, bucketID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		membership, err := GetUserMembership(gormDB, userID, bucketID)

		require.NoError(t, err)
		assert.Nil(t, membership)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should return error on database failure", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(userID, bucketID, 1).
			WillReturnError(sql.ErrConnDone)

		membership, err := GetUserMembership(gormDB, userID, bucketID)

		assert.Error(t, err)
		assert.Nil(t, membership)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestGetBucketMembers tests retrieving all members of a bucket
func TestGetBucketMembers(t *testing.T) {
	t.Run("should return all bucket members", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		bucketID := uuid.New()
		user1ID := uuid.New()
		user2ID := uuid.New()
		membership1ID := uuid.New()
		membership2ID := uuid.New()

		// Mock the membership query with soft delete check
		membershipRows := sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group", "created_at", "updated_at", "deleted_at"}).
			AddRow(membership1ID, user1ID, bucketID, "owner", nil, nil, nil).
			AddRow(membership2ID, user2ID, bucketID, "viewer", nil, nil, nil)

		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(bucketID).
			WillReturnRows(membershipRows)

		// Mock the User preload query
		userRows := sqlmock.NewRows([]string{"id", "email", "role", "first_name", "last_name", "hashed_password", "is_initialized", "provider_type", "provider_key", "created_at", "updated_at", "deleted_at"}).
			AddRow(user1ID, "owner@example.com", "user", "", "", "", false, "local", "", nil, nil, nil).
			AddRow(user2ID, "viewer@example.com", "user", "", "", "", false, "local", "", nil, nil, nil)

		mock.ExpectQuery(`SELECT \* FROM "users"`).
			WillReturnRows(userRows)

		memberships, err := GetBucketMembers(gormDB, bucketID)

		require.NoError(t, err)
		assert.Len(t, memberships, 2)
		assert.Equal(t, models.GroupOwner, memberships[0].Group)
		assert.Equal(t, models.GroupViewer, memberships[1].Group)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should return empty slice when no members", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		bucketID := uuid.New()

		membershipRows := sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group", "created_at", "updated_at", "deleted_at"})

		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(bucketID).
			WillReturnRows(membershipRows)

		memberships, err := GetBucketMembers(gormDB, bucketID)

		require.NoError(t, err)
		assert.Empty(t, memberships)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestGetUserBuckets tests retrieving all buckets a user has access to
func TestGetUserBuckets(t *testing.T) {
	t.Run("should return all user buckets", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucket1ID := uuid.New()
		bucket2ID := uuid.New()
		membership1ID := uuid.New()
		membership2ID := uuid.New()

		// Mock the membership query with soft delete check
		membershipRows := sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group", "created_at", "updated_at", "deleted_at"}).
			AddRow(membership1ID, userID, bucket1ID, "owner", nil, nil, nil).
			AddRow(membership2ID, userID, bucket2ID, "contributor", nil, nil, nil)

		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(userID).
			WillReturnRows(membershipRows)

		// Mock the Bucket preload query
		bucketRows := sqlmock.NewRows([]string{"id", "name", "created_by", "created_at", "updated_at", "deleted_at"}).
			AddRow(bucket1ID, "My Bucket", userID, nil, nil, nil).
			AddRow(bucket2ID, "Shared Bucket", uuid.New(), nil, nil, nil)

		mock.ExpectQuery(`SELECT \* FROM "buckets"`).
			WillReturnRows(bucketRows)

		memberships, err := GetUserBuckets(gormDB, userID)

		require.NoError(t, err)
		assert.Len(t, memberships, 2)
		assert.Equal(t, models.GroupOwner, memberships[0].Group)
		assert.Equal(t, models.GroupContributor, memberships[1].Group)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestCreateMembership tests creating a new membership
func TestCreateMembership(t *testing.T) {
	t.Run("should create membership successfully", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()
		group := models.GroupOwner

		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "memberships"`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
		mock.ExpectCommit()

		err := CreateMembership(gormDB, userID, bucketID, group)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should return error on database failure", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()
		group := models.GroupOwner

		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "memberships"`).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		err := CreateMembership(gormDB, userID, bucketID, group)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUpdateMembership tests updating a membership's group
func TestUpdateMembership(t *testing.T) {
	t.Run("should update membership group successfully", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()
		newGroup := models.GroupContributor

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "memberships"`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := UpdateMembership(gormDB, userID, bucketID, newGroup)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should return error on database failure", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()
		newGroup := models.GroupContributor

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "memberships"`).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		err := UpdateMembership(gormDB, userID, bucketID, newGroup)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestDeleteMembership tests deleting a membership
func TestDeleteMembership(t *testing.T) {
	t.Run("should delete membership successfully", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "memberships"`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := DeleteMembership(gormDB, userID, bucketID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should return error on database failure", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "memberships"`).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		err := DeleteMembership(gormDB, userID, bucketID)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestHasBucketAccess tests checking if a user has bucket access
func TestHasBucketAccess(t *testing.T) {
	t.Run("should return true when user has sufficient access", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()
		membershipID := uuid.New()

		// User is an owner, checking for contributor access
		rows := sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group", "created_at", "updated_at", "deleted_at"}).
			AddRow(membershipID, userID, bucketID, "owner", nil, nil, nil)

		// GORM adds LIMIT 1 for First() queries
		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(userID, bucketID, 1).
			WillReturnRows(rows)

		hasAccess, err := HasBucketAccess(gormDB, userID, bucketID, models.GroupContributor)

		require.NoError(t, err)
		assert.True(t, hasAccess, "Owner should have contributor access")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should return false when user has insufficient access", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()
		membershipID := uuid.New()

		// User is a viewer, checking for owner access
		rows := sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group", "created_at", "updated_at", "deleted_at"}).
			AddRow(membershipID, userID, bucketID, "viewer", nil, nil, nil)

		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(userID, bucketID, 1).
			WillReturnRows(rows)

		hasAccess, err := HasBucketAccess(gormDB, userID, bucketID, models.GroupOwner)

		require.NoError(t, err)
		assert.False(t, hasAccess, "Viewer should NOT have owner access (privilege escalation)")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should return false when user has no membership", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(userID, bucketID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		hasAccess, err := HasBucketAccess(gormDB, userID, bucketID, models.GroupViewer)

		require.NoError(t, err)
		assert.False(t, hasAccess, "User without membership should not have access")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should return error on database failure", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(userID, bucketID, 1).
			WillReturnError(sql.ErrConnDone)

		hasAccess, err := HasBucketAccess(gormDB, userID, bucketID, models.GroupViewer)

		assert.Error(t, err)
		assert.False(t, hasAccess)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestHasBucketAccess_SecurityScenarios tests security-critical access control scenarios
func TestHasBucketAccess_SecurityScenarios(t *testing.T) {
	t.Run("prevent privilege escalation from viewer to owner", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()
		membershipID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group", "created_at", "updated_at", "deleted_at"}).
			AddRow(membershipID, userID, bucketID, "viewer", nil, nil, nil)

		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(userID, bucketID, 1).
			WillReturnRows(rows)

		hasAccess, err := HasBucketAccess(gormDB, userID, bucketID, models.GroupOwner)

		require.NoError(t, err)
		assert.False(t, hasAccess, "Viewer must not escalate to owner")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("prevent privilege escalation from contributor to owner", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()
		membershipID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group", "created_at", "updated_at", "deleted_at"}).
			AddRow(membershipID, userID, bucketID, "contributor", nil, nil, nil)

		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(userID, bucketID, 1).
			WillReturnRows(rows)

		hasAccess, err := HasBucketAccess(gormDB, userID, bucketID, models.GroupOwner)

		require.NoError(t, err)
		assert.False(t, hasAccess, "Contributor must not escalate to owner")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("owner can downgrade to lower permissions", func(t *testing.T) {
		gormDB, mock, db := setupMockDB(t)
		defer db.Close()

		userID := uuid.New()
		bucketID := uuid.New()
		membershipID := uuid.New()

		// Test owner acting as viewer
		rows := sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group", "created_at", "updated_at", "deleted_at"}).
			AddRow(membershipID, userID, bucketID, "owner", nil, nil, nil)

		mock.ExpectQuery(`SELECT \* FROM "memberships"`).
			WithArgs(userID, bucketID, 1).
			WillReturnRows(rows)

		hasAccess, err := HasBucketAccess(gormDB, userID, bucketID, models.GroupViewer)

		require.NoError(t, err)
		assert.True(t, hasAccess, "Owner can perform viewer actions")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
