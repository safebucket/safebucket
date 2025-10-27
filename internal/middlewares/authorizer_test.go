package middlewares

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"api/internal/models"
	"api/internal/tests"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// mockNextHandler is a simple handler that returns 200 OK.
func mockAuthNextHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func TestAuthorizeRole(t *testing.T) {
	testCases := []struct {
		name           string
		userRole       models.Role
		requiredRole   models.Role
		hasUserClaims  bool
		expectedStatus int
		expectedErrors []string
	}{
		{
			name:           "Admin accessing User-required endpoint",
			userRole:       models.RoleAdmin,
			requiredRole:   models.RoleUser,
			hasUserClaims:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User accessing User-required endpoint",
			userRole:       models.RoleUser,
			requiredRole:   models.RoleUser,
			hasUserClaims:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Admin accessing Admin-required endpoint",
			userRole:       models.RoleAdmin,
			requiredRole:   models.RoleAdmin,
			hasUserClaims:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Guest accessing User-required endpoint",
			userRole:       models.RoleGuest,
			requiredRole:   models.RoleUser,
			hasUserClaims:  true,
			expectedStatus: http.StatusForbidden,
			expectedErrors: []string{"FORBIDDEN"},
		},
		{
			name:           "User accessing Admin-required endpoint",
			userRole:       models.RoleUser,
			requiredRole:   models.RoleAdmin,
			hasUserClaims:  true,
			expectedStatus: http.StatusForbidden,
			expectedErrors: []string{"FORBIDDEN"},
		},
		{
			name:           "Missing user claims in context",
			hasUserClaims:  false,
			requiredRole:   models.RoleUser,
			expectedStatus: http.StatusUnauthorized,
			expectedErrors: []string{"UNAUTHORIZED"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			if tt.hasUserClaims {
				userClaims := models.UserClaims{
					UserID: uuid.New(),
					Email:  "test@example.com",
					Role:   tt.userRole,
				}
				ctx := context.WithValue(req.Context(), models.UserClaimKey{}, userClaims)
				req = req.WithContext(ctx)
			}

			handler := AuthorizeRole(tt.requiredRole)(http.HandlerFunc(mockAuthNextHandler))
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code)

			if tt.expectedStatus != http.StatusOK {
				expected := models.Error{Status: tt.expectedStatus, Error: tt.expectedErrors}
				tests.AssertJSONResponse(t, recorder, tt.expectedStatus, expected)
			}
		})
	}
}

func TestAuthorizeGroup(t *testing.T) {
	userID := uuid.New()
	bucketID := uuid.New()

	testCases := []struct {
		name             string
		userGroup        models.Group
		requiredGroup    models.Group
		hasUserClaims    bool
		bucketIDIndex    int
		setupURLParams   bool
		hasMembership    bool
		mockDBError      bool
		expectedStatus   int
		expectedErrors   []string
		setupMockQueries func(sqlmock.Sqlmock)
	}{
		{
			name:           "Owner accessing Viewer-required endpoint",
			userGroup:      models.GroupOwner,
			requiredGroup:  models.GroupViewer,
			hasUserClaims:  true,
			bucketIDIndex:  0,
			setupURLParams: true,
			hasMembership:  true,
			expectedStatus: http.StatusOK,
			setupMockQueries: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group"}).
					AddRow(uuid.New(), userID, bucketID, models.GroupOwner)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "memberships" WHERE (user_id = $1 AND bucket_id = $2) AND "memberships"."deleted_at" IS NULL ORDER BY "memberships"."id" LIMIT $3`)).
					WithArgs(userID, bucketID, 1).
					WillReturnRows(rows)
			},
		},
		{
			name:           "Contributor accessing Contributor-required endpoint",
			userGroup:      models.GroupContributor,
			requiredGroup:  models.GroupContributor,
			hasUserClaims:  true,
			bucketIDIndex:  0,
			setupURLParams: true,
			hasMembership:  true,
			expectedStatus: http.StatusOK,
			setupMockQueries: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group"}).
					AddRow(uuid.New(), userID, bucketID, models.GroupContributor)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "memberships" WHERE (user_id = $1 AND bucket_id = $2) AND "memberships"."deleted_at" IS NULL ORDER BY "memberships"."id" LIMIT $3`)).
					WithArgs(userID, bucketID, 1).
					WillReturnRows(rows)
			},
		},
		{
			name:           "Viewer accessing Owner-required endpoint",
			userGroup:      models.GroupViewer,
			requiredGroup:  models.GroupOwner,
			hasUserClaims:  true,
			bucketIDIndex:  0,
			setupURLParams: true,
			hasMembership:  true,
			expectedStatus: http.StatusForbidden,
			expectedErrors: []string{"FORBIDDEN"},
			setupMockQueries: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group"}).
					AddRow(uuid.New(), userID, bucketID, models.GroupViewer)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "memberships" WHERE (user_id = $1 AND bucket_id = $2) AND "memberships"."deleted_at" IS NULL ORDER BY "memberships"."id" LIMIT $3`)).
					WithArgs(userID, bucketID, 1).
					WillReturnRows(rows)
			},
		},
		{
			name:           "User with no membership",
			requiredGroup:  models.GroupViewer,
			hasUserClaims:  true,
			bucketIDIndex:  0,
			setupURLParams: true,
			hasMembership:  false,
			expectedStatus: http.StatusForbidden,
			expectedErrors: []string{"FORBIDDEN"},
			setupMockQueries: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "memberships" WHERE (user_id = $1 AND bucket_id = $2) AND "memberships"."deleted_at" IS NULL ORDER BY "memberships"."id" LIMIT $3`)).
					WithArgs(userID, bucketID, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "bucket_id", "group"}))
			},
		},
		{
			name:           "Missing user claims",
			requiredGroup:  models.GroupViewer,
			hasUserClaims:  false,
			bucketIDIndex:  0,
			setupURLParams: true,
			expectedStatus: http.StatusUnauthorized,
			expectedErrors: []string{"UNAUTHORIZED"},
			setupMockQueries: func(_ sqlmock.Sqlmock) {
			},
		},
		{
			name:           "Invalid bucket UUID in URL",
			requiredGroup:  models.GroupViewer,
			hasUserClaims:  true,
			bucketIDIndex:  0,
			setupURLParams: false,
			expectedStatus: http.StatusUnauthorized,
			expectedErrors: []string{"UNAUTHORIZED"},
			setupMockQueries: func(_ sqlmock.Sqlmock) {
			},
		},
		{
			name:           "Database error",
			requiredGroup:  models.GroupViewer,
			hasUserClaims:  true,
			bucketIDIndex:  0,
			setupURLParams: true,
			mockDBError:    true,
			expectedStatus: http.StatusInternalServerError,
			expectedErrors: []string{"INTERNAL_SERVER_ERROR"},
			setupMockQueries: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "memberships" WHERE (user_id = $1 AND bucket_id = $2) AND "memberships"."deleted_at" IS NULL ORDER BY "memberships"."id" LIMIT $3`)).
					WithArgs(userID, bucketID, 1).
					WillReturnError(gorm.ErrInvalidDB)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func(db *sql.DB) {
				_ = db.Close()
			}(db)

			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: db,
			}), &gorm.Config{})
			require.NoError(t, err)

			tt.setupMockQueries(mock)

			req := httptest.NewRequest(http.MethodGet, "/buckets/"+bucketID.String(), nil)
			recorder := httptest.NewRecorder()

			if tt.hasUserClaims {
				userClaims := models.UserClaims{
					UserID: userID,
					Email:  "test@example.com",
					Role:   models.RoleUser,
				}
				ctx := context.WithValue(req.Context(), models.UserClaimKey{}, userClaims)
				req = req.WithContext(ctx)
			}

			if tt.setupURLParams {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id0", bucketID.String())
				ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
				req = req.WithContext(ctx)
			}

			handler := AuthorizeGroup(
				gormDB,
				tt.requiredGroup,
				tt.bucketIDIndex,
			)(
				http.HandlerFunc(mockAuthNextHandler),
			)
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code)

			if tt.expectedStatus != http.StatusOK {
				expected := models.Error{Status: tt.expectedStatus, Error: tt.expectedErrors}
				tests.AssertJSONResponse(t, recorder, tt.expectedStatus, expected)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAuthorizeSelfOrAdmin(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()

	testCases := []struct {
		name            string
		authenticatedID uuid.UUID
		targetID        uuid.UUID
		userRole        models.Role
		hasUserClaims   bool
		targetUserIDIdx int
		setupURLParams  bool
		expectedStatus  int
		expectedErrors  []string
	}{
		{
			name:            "User accessing own resource",
			authenticatedID: userID,
			targetID:        userID,
			userRole:        models.RoleUser,
			hasUserClaims:   true,
			targetUserIDIdx: 0,
			setupURLParams:  true,
			expectedStatus:  http.StatusOK,
		},
		{
			name:            "Admin accessing other user's resource",
			authenticatedID: userID,
			targetID:        otherUserID,
			userRole:        models.RoleAdmin,
			hasUserClaims:   true,
			targetUserIDIdx: 0,
			setupURLParams:  true,
			expectedStatus:  http.StatusOK,
		},
		{
			name:            "Non-admin user accessing other user's resource",
			authenticatedID: userID,
			targetID:        otherUserID,
			userRole:        models.RoleUser,
			hasUserClaims:   true,
			targetUserIDIdx: 0,
			setupURLParams:  true,
			expectedStatus:  http.StatusForbidden,
			expectedErrors:  []string{"FORBIDDEN"},
		},
		{
			name:            "Guest accessing other user's resource",
			authenticatedID: userID,
			targetID:        otherUserID,
			userRole:        models.RoleGuest,
			hasUserClaims:   true,
			targetUserIDIdx: 0,
			setupURLParams:  true,
			expectedStatus:  http.StatusForbidden,
			expectedErrors:  []string{"FORBIDDEN"},
		},
		{
			name:            "Missing user claims",
			hasUserClaims:   false,
			targetUserIDIdx: 0,
			setupURLParams:  true,
			expectedStatus:  http.StatusUnauthorized,
			expectedErrors:  []string{"UNAUTHORIZED"},
		},
		{
			name:            "Invalid target user UUID",
			authenticatedID: userID,
			userRole:        models.RoleUser,
			hasUserClaims:   true,
			targetUserIDIdx: 0,
			setupURLParams:  false,
			expectedStatus:  http.StatusUnauthorized,
			expectedErrors:  []string{"UNAUTHORIZED"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			urlPath := "/users/" + tt.targetID.String()
			if !tt.setupURLParams {
				urlPath = "/users/invalid-uuid"
			}

			req := httptest.NewRequest(http.MethodGet, urlPath, nil)
			recorder := httptest.NewRecorder()

			if tt.hasUserClaims {
				userClaims := models.UserClaims{
					UserID: tt.authenticatedID,
					Email:  "test@example.com",
					Role:   tt.userRole,
				}
				ctx := context.WithValue(req.Context(), models.UserClaimKey{}, userClaims)
				req = req.WithContext(ctx)
			}

			if tt.setupURLParams {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id0", tt.targetID.String())
				ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
				req = req.WithContext(ctx)
			}

			handler := AuthorizeSelfOrAdmin(
				tt.targetUserIDIdx,
			)(
				http.HandlerFunc(mockAuthNextHandler),
			)
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code)

			if tt.expectedStatus != http.StatusOK {
				expected := models.Error{Status: tt.expectedStatus, Error: tt.expectedErrors}
				tests.AssertJSONResponse(t, recorder, tt.expectedStatus, expected)
			}
		})
	}
}
