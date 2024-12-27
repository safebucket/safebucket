package handlers

import (
	customErr "api/internal/errors"
	h "api/internal/helpers"
	"api/internal/models"
	"api/internal/tests"
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateHandler(t *testing.T) {
	mockInput := models.Bucket{Name: "Bucket1"}
	mockOutput := models.Bucket{ID: uuid.New(), Name: "John Doe"}

	mockCreate := new(tests.MockCreateFunc[models.Bucket, models.Bucket])
	mockCreate.On("Create", uuid.UUIDs(nil), mockInput).Return(mockOutput, nil)

	req := httptest.NewRequest(http.MethodPost, "/buckets", nil)
	recorder := httptest.NewRecorder()

	req = req.WithContext(context.WithValue(req.Context(), h.BodyKey{}, mockInput))

	handler := CreateHandler(mockCreate.Create)
	handler(recorder, req)

	mockCreate.AssertExpectations(t)
	tests.AssertJSONResponse(t, recorder, http.StatusCreated, mockOutput)
}

func TestCreateHandler_BadRequest(t *testing.T) {
	mockInput := models.Bucket{Name: "Bucket1"}

	mockCreate := new(tests.MockCreateFunc[models.Bucket, models.Bucket])
	mockCreate.On(
		"Create",
		uuid.UUIDs(nil),
		mockInput,
	).Return(models.Bucket{}, errors.New("INVALID_DATA"))

	req := httptest.NewRequest(http.MethodPost, "/buckets", nil)
	recorder := httptest.NewRecorder()

	req = req.WithContext(context.WithValue(req.Context(), h.BodyKey{}, mockInput))

	handler := CreateHandler(mockCreate.Create)
	handler(recorder, req)

	mockCreate.AssertExpectations(t)
	expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_DATA"}}
	tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
}

func TestGetListHandler(t *testing.T) {
	mockGetListFunc := new(tests.MockGetListFunc[models.Bucket])

	records := []models.Bucket{
		{
			ID:        uuid.UUID{},
			Name:      "bucket1",
			CreatedAt: time.Time{},
		},
		{
			ID:        uuid.UUID{},
			Name:      "bucket2",
			CreatedAt: time.Time{},
		},
	}

	mockGetListFunc.On("GetList").Return(records)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/buckets", nil)

	handler := GetListHandler(mockGetListFunc.GetList)
	handler(recorder, req)

	mockGetListFunc.AssertExpectations(t)
	page := models.Page[models.Bucket]{Data: records}
	tests.AssertJSONResponse(t, recorder, http.StatusOK, page)
}

func TestGetOneHandler(t *testing.T) {
	testUUID := uuid.New()

	expectedRecord := models.Bucket{
		ID:   testUUID,
		Name: "bucket1",
	}

	mockGetOne := new(tests.MockGetOneFunc[models.Bucket])
	mockGetOne.On("GetOne", uuid.UUIDs{testUUID}).Return(expectedRecord, nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := GetOneHandler(mockGetOne.GetOne)
	handler(recorder, req)

	mockGetOne.AssertExpectations(t)
	tests.AssertJSONResponse(t, recorder, http.StatusOK, expectedRecord)
}

func TestGetOneHandler_NotFound(t *testing.T) {
	testUUID := uuid.New()

	mockGetOne := new(tests.MockGetOneFunc[models.Bucket])
	mockGetOne.On("GetOne", uuid.UUIDs{testUUID}).Return(models.Bucket{}, errors.New("RECORD_NOT_FOUND"))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := GetOneHandler(mockGetOne.GetOne)
	handler(recorder, req)

	mockGetOne.AssertExpectations(t)
	expected := models.Error{Status: http.StatusNotFound, Error: []string{"RECORD_NOT_FOUND"}}
	tests.AssertJSONResponse(t, recorder, http.StatusNotFound, expected)
}

func TestGetOneHandler_InvalidUUID(t *testing.T) {
	invalidUUID := "invalid"
	mockGetOne := new(tests.MockGetOneFunc[models.Bucket])

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/buckets/%s", invalidUUID), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", invalidUUID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := GetOneHandler(mockGetOne.GetOne)
	handler(recorder, req)

	mockGetOne.AssertExpectations(t)
	expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_UUID"}}
	tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
}

func TestUpdateHandler(t *testing.T) {
	testUUID := uuid.New()
	mockInput := models.Bucket{Name: "New name"}
	mockOutput := models.Bucket{ID: testUUID, Name: "New Name"}

	mockUpdate := new(tests.MockUpdateFunc[models.Bucket, models.Bucket])
	mockUpdate.On("Update", uuid.UUIDs{testUUID}, mockInput).Return(mockOutput, nil)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), h.BodyKey{}, mockInput))

	handler := UpdateHandler(mockUpdate.Update)
	handler(recorder, req)

	mockUpdate.AssertExpectations(t)
	tests.AssertJSONResponse(t, recorder, http.StatusNoContent, nil)
}

func TestUpdateHandler_InvalidUUID(t *testing.T) {
	invalidUUID := "invalid"
	mockUpdate := new(tests.MockUpdateFunc[models.Bucket, models.Bucket])

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/buckets/%s", invalidUUID), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", invalidUUID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := UpdateHandler(mockUpdate.Update)
	handler(recorder, req)

	mockUpdate.AssertExpectations(t)
	expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_UUID"}}
	tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
}

func TestUpdateHandler_NotFound(t *testing.T) {
	testUUID := uuid.New()
	mockInput := models.Bucket{Name: "Updated Name"}

	mockUpdate := new(tests.MockUpdateFunc[models.Bucket, models.Bucket])
	mockUpdate.On(
		"Update",
		uuid.UUIDs{testUUID},
		mockInput,
	).Return(models.Bucket{}, customErr.NewAPIError(404, "NOT_FOUND"))

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), h.BodyKey{}, mockInput))

	handler := UpdateHandler(mockUpdate.Update)
	handler(recorder, req)

	mockUpdate.AssertExpectations(t)
	expected := models.Error{Status: http.StatusNotFound, Error: []string{"NOT_FOUND"}}
	tests.AssertJSONResponse(t, recorder, http.StatusNotFound, expected)
}

func TestDeleteHandler(t *testing.T) {
	testUUID := uuid.New()

	mockDelete := new(tests.MockDeleteFunc)
	mockDelete.On("Delete", uuid.UUIDs{testUUID}).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := DeleteHandler(mockDelete.Delete)
	handler(recorder, req)

	mockDelete.AssertExpectations(t)
	tests.AssertJSONResponse(t, recorder, http.StatusNoContent, nil)
}

func TestDeleteHandler_NotFound(t *testing.T) {
	testUUID := uuid.New()

	mockDelete := new(tests.MockDeleteFunc)
	mockDelete.On("Delete", uuid.UUIDs{testUUID}).Return(errors.New("RECORD_NOT_FOUND"))

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := DeleteHandler(mockDelete.Delete)
	handler(recorder, req)

	mockDelete.AssertExpectations(t)
	expected := models.Error{Status: http.StatusNotFound, Error: []string{"RECORD_NOT_FOUND"}}
	tests.AssertJSONResponse(t, recorder, http.StatusNotFound, expected)
}

func TestDeleteHandler_InvalidUUID(t *testing.T) {
	invalidUUID := "invalid"

	mockDelete := new(tests.MockDeleteFunc)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/buckets/%s", invalidUUID), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", invalidUUID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := DeleteHandler(mockDelete.Delete)
	handler(recorder, req)

	mockDelete.AssertExpectations(t)
	expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_UUID"}}
	tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
}
