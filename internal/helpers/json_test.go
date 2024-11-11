package helpers

import (
	"api/internal/models"
	"api/internal/tests"
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseUUID(t *testing.T) {
	t.Run("Valid UUID", func(t *testing.T) {
		expectedUUID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/test/%s", expectedUUID.String()), nil)
		recorder := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", expectedUUID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		parsedUUID, ok := ParseUUID(recorder, req)

		assert.True(t, ok)
		assert.Equal(t, expectedUUID, parsedUUID)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		invalidUUID := "not-a-uuid"
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/test/%s", invalidUUID), nil)
		recorder := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", invalidUUID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		parsedUUID, ok := ParseUUID(recorder, req)

		expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_UUID"}}
		tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
		assert.False(t, ok)
		assert.Equal(t, uuid.Nil, parsedUUID)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Missing UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test/", nil)
		recorder := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		parsedUUID, ok := ParseUUID(recorder, req)

		expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_UUID"}}
		tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
		assert.False(t, ok)
		assert.Equal(t, uuid.Nil, parsedUUID)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestRespondWithJSON_List(t *testing.T) {
	recorder := httptest.NewRecorder()

	payload := []models.Bucket{
		{
			ID:        uuid.UUID{},
			Name:      "file1.txt",
			CreatedAt: time.Time{},
		},
		{
			ID:        uuid.UUID{},
			Name:      "file2.pdf",
			CreatedAt: time.Time{},
		},
	}

	RespondWithJSON(recorder, http.StatusOK, payload)

	tests.AssertJSONResponse(t, recorder, http.StatusOK, payload)
}

func TestRespondWithJSON_Detail(t *testing.T) {
	recorder := httptest.NewRecorder()

	payload := models.Bucket{
		ID:        uuid.UUID{},
		Name:      "file1.txt",
		CreatedAt: time.Time{},
	}

	RespondWithJSON(recorder, http.StatusOK, payload)

	tests.AssertJSONResponse(t, recorder, http.StatusOK, payload)
}

func TestRespondWithJSON_NoContent(t *testing.T) {
	recorder := httptest.NewRecorder()

	RespondWithJSON(recorder, http.StatusNoContent, nil)

	tests.AssertJSONResponse(t, recorder, http.StatusNoContent, nil)
}

func TestRespondWithJSON_JSONMarshallingError(t *testing.T) {
	recorder := httptest.NewRecorder()
	payload := make(chan int)

	RespondWithJSON(recorder, http.StatusInternalServerError, payload)

	tests.AssertJSONResponse(t, recorder, http.StatusInternalServerError, nil)
}

func TestRespondWithError(t *testing.T) {
	recorder := httptest.NewRecorder()
	errorMessage := []string{"invalid input"}

	RespondWithError(recorder, http.StatusBadRequest, errorMessage)

	expected := models.Error{Status: http.StatusBadRequest, Error: errorMessage}
	tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
}
