//go:build integration

package integration

import (
	"testing"

	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func CreateTestUser(t *testing.T, db *gorm.DB, email string, role models.Role) models.User {
	t.Helper()

	hash, err := helpers.CreateHash(testPassword)
	require.NoError(t, err)

	user := models.User{
		FirstName:      "Test",
		LastName:       "User",
		Email:          email,
		HashedPassword: hash,
		IsInitialized:  true,
		ProviderType:   models.LocalProviderType,
		ProviderKey:    string(models.LocalProviderType),
		Role:           role,
	}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func CreateTestBucket(t *testing.T, db *gorm.DB, name string, ownerID uuid.UUID) models.Bucket {
	t.Helper()

	bucket := models.Bucket{Name: name, CreatedBy: ownerID}
	require.NoError(t, db.Create(&bucket).Error)

	membership := models.Membership{
		UserID:   ownerID,
		BucketID: bucket.ID,
		Group:    models.GroupOwner,
	}
	require.NoError(t, db.Create(&membership).Error)

	return bucket
}

func CreateTestInvite(
	t *testing.T,
	db *gorm.DB,
	email string,
	bucketID uuid.UUID,
	group models.Group,
	createdBy uuid.UUID,
) models.Invite {
	t.Helper()

	invite := models.Invite{
		Email:     email,
		Group:     group,
		BucketID:  bucketID,
		CreatedBy: createdBy,
	}
	require.NoError(t, db.Create(&invite).Error)
	return invite
}
