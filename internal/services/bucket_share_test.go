package services

import (
	"errors"
	"net/http"
	"testing"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCreateShareTransactionFailure(t *testing.T) {
	testCases := []struct {
		name           string
		customID       string
		matchingShares int64
		lookupError    error
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "custom ID taken after availability check", customID: "project-files", matchingShares: 1,
			expectedStatus: http.StatusConflict, expectedCode: apierrors.CodeShareCustomIDAlreadyExists,
		},
		{
			name: "unrelated transaction failure", customID: "project-files",
			expectedStatus: http.StatusInternalServerError, expectedCode: apierrors.CodeInternalServerError,
		},
		{
			name: "post-failure lookup fails", customID: "project-files", lookupError: errors.New("lookup failed"),
			expectedStatus: http.StatusInternalServerError, expectedCode: apierrors.CodeInternalServerError,
		},
		{
			name:           "standard share transaction failure",
			expectedStatus: http.StatusInternalServerError, expectedCode: apierrors.CodeInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = sqlDB.Close() })
			db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
			require.NoError(t, err)

			bucketID := uuid.New()
			if tc.customID != "" {
				mock.ExpectQuery(`SELECT .* FROM "shares" WHERE custom_id = \$1`).WithArgs(tc.customID).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))
			}
			mock.ExpectQuery(`SELECT .* FROM "buckets" WHERE id = \$1`).WithArgs(bucketID).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(bucketID))
			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "shares"`).WillReturnError(errors.New("insert failed"))
			mock.ExpectRollback()
			if tc.customID != "" {
				lookup := mock.ExpectQuery(`SELECT count\(\*\) FROM "shares" WHERE custom_id = \$1`).
					WithArgs(tc.customID)
				if tc.lookupError != nil {
					lookup.WillReturnError(tc.lookupError)
				} else {
					lookup.WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tc.matchingShares))
				}
			}

			service := BucketShareService{DB: db, AllowCustomShareIDs: true}
			share, err := service.CreateShare(zap.NewNop(), models.UserClaims{}, uuid.UUIDs{bucketID},
				models.ShareCreateBody{CustomID: tc.customID, Name: "project", Type: models.ShareTypeBucket})
			var apiErr *apierrors.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.expectedStatus, apiErr.Status)
			assert.Equal(t, tc.expectedCode, apiErr.Code)
			assert.Empty(t, share)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
