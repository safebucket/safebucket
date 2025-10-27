package helpers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api/internal/models"
	"api/internal/tests"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestParseUUID(t *testing.T) {
	t.Run("Valid UUID", func(t *testing.T) {
		expectedUUID := uuid.New()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/test/%s", expectedUUID.String()),
			nil,
		)
		recorder := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id0", expectedUUID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		uuids, ok := ParseUUIDs(recorder, req)

		assert.True(t, ok)
		assert.Equal(t, uuid.UUIDs{expectedUUID}, uuids)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("Multiple valid UUIDs", func(t *testing.T) {
		expectedUUID := uuid.New()
		expectedUUID2 := uuid.New()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/test/%s/test/%s", expectedUUID.String(), expectedUUID2.String()),
			nil,
		)
		recorder := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id0", expectedUUID.String())
		rctx.URLParams.Add("id1", expectedUUID2.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		uuids, ok := ParseUUIDs(recorder, req)

		assert.True(t, ok)
		assert.Equal(t, uuid.UUIDs{expectedUUID, expectedUUID2}, uuids)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		invalidUUID := "not-a-uuid"
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/test/%s", invalidUUID), nil)
		recorder := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id0", invalidUUID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		uuids, ok := ParseUUIDs(recorder, req)

		expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_UUID"}}
		tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
		assert.False(t, ok)
		assert.Equal(t, uuid.UUIDs(nil), uuids)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Multiple UUIDs with one invalid", func(t *testing.T) {
		invalidUUID := "not-a-uuid"
		expectedUUID := uuid.New()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/test/%s/test/%s", expectedUUID, invalidUUID),
			nil,
		)
		recorder := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id0", expectedUUID.String())
		rctx.URLParams.Add("id1", invalidUUID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		uuids, ok := ParseUUIDs(recorder, req)

		expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_UUID"}}
		tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
		assert.False(t, ok)
		assert.Equal(t, uuid.UUIDs{expectedUUID}, uuids)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("More UUIDs than hard limit", func(t *testing.T) {
		expectedUUID := uuid.New()
		expectedUUID2 := uuid.New()
		extraUUID := uuid.New()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/test/%s/test/%s/test", expectedUUID, expectedUUID2),
			nil,
		)
		recorder := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id0", expectedUUID.String())
		rctx.URLParams.Add("id1", expectedUUID2.String())
		rctx.URLParams.Add("id2", extraUUID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		uuids, ok := ParseUUIDs(recorder, req)

		assert.True(t, ok)
		assert.Equal(t, uuid.UUIDs{expectedUUID, expectedUUID2}, uuids)
		assert.Equal(t, http.StatusOK, recorder.Code)
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
