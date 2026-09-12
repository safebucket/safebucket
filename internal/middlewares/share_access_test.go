package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apierrors "github.com/safebucket/safebucket/internal/errors"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateShareAccessInvalidID(t *testing.T) {
	testCases := []struct {
		name    string
		shareID string
	}{
		{name: "empty", shareID: ""},
		{name: "too short", shareID: "ab"},
		{name: "too long", shareID: strings.Repeat("a", 256)},
		{name: "slash", shareID: "project/files"},
		{name: "space", shareID: "project files"},
		{name: "unicode", shareID: "projet-été"},
		{name: "newline", shareID: "project\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/shares/test", nil)
			routeContext := chi.NewRouteContext()
			routeContext.URLParams.Add("shareId", tc.shareID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
			recorder := httptest.NewRecorder()
			handler := ValidateShareAccess(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("invalid share ID reached the next handler")
			}))
			handler.ServeHTTP(recorder, req)
			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.Contains(t, recorder.Body.String(), apierrors.CodeInvalidValue)
		})
	}
}

func TestValidateShareAccessValidID(t *testing.T) {
	const (
		customIDQuery = `WHERE custom_id = \$1`
		shareIDQuery  = `custom_id IS NULL`
	)

	testCases := []struct {
		name         string
		shareID      string
		expectedSQL  string
		expectedRows bool
	}{
		{name: "minimum length", shareID: "a_-", expectedSQL: customIDQuery, expectedRows: true},
		{
			name: "maximum length", shareID: strings.Repeat("a", 255),
			expectedSQL: customIDQuery, expectedRows: true,
		},
		{name: "mixed case", shareID: "Project_files-2026", expectedSQL: customIDQuery, expectedRows: true},
		{name: "canonical UUID", shareID: uuid.NewString(), expectedSQL: shareIDQuery, expectedRows: true},
		{
			name: "compact UUID", shareID: "550e8400e29b41d4a716446655440000",
			expectedSQL: customIDQuery, expectedRows: false,
		},
		{name: "missing share", shareID: "missing-share", expectedSQL: customIDQuery, expectedRows: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := newGormWithMock(t)
			defer sqlDB.Close()
			rows := sqlmock.NewRows([]string{"id"})
			if tc.expectedRows {
				rows.AddRow(uuid.NewString())
			}
			mock.ExpectQuery(tc.expectedSQL).WithArgs(tc.shareID).WillReturnRows(rows)

			router := chi.NewRouter()
			router.With(ValidateShareAccess(db)).Get("/{shareId}", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/"+tc.shareID, nil))
			expectedStatus := http.StatusNotFound
			if tc.expectedRows {
				expectedStatus = http.StatusNoContent
			}
			assert.Equal(t, expectedStatus, recorder.Code)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
