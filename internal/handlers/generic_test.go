package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestCreateHandler(t *testing.T) {
	testUUID := uuid.New()
	mockInput := models.BucketCreateUpdateBody{Name: "test-bucket"}
	mockOutput := models.Bucket{
		ID:        testUUID,
		Name:      "test-bucket",
		CreatedAt: time.Now(),
	}

	mockCreate := new(tests.MockCreateFunc[models.BucketCreateUpdateBody, models.Bucket])
	mockCreate.On(
		"Create",
		mock.AnythingOfType("*zap.Logger"),
		mock.Anything, // claims
		uuid.UUIDs(nil),
		mockInput,
	).Return(mockOutput, nil)

	req := httptest.NewRequest(http.MethodPost, "/buckets", nil)
	recorder := httptest.NewRecorder()

	logger := zap.NewNop()
	ctx := context.WithValue(req.Context(), m.LoggerKey, logger)
	claims := models.UserClaims{
		UserID: uuid.New(),
		Email:  "test@example.com",
		Role:   models.RoleUser,
	}
	ctx = context.WithValue(ctx, models.UserClaimKey{}, claims)
	ctx = context.WithValue(ctx, m.BodyKey{}, mockInput)
	req = req.WithContext(ctx)

	handler := CreateHandler(mockCreate.Create)
	handler(recorder, req)

	mockCreate.AssertExpectations(t)
	tests.AssertJSONResponse(t, recorder, http.StatusCreated, mockOutput)
}

func TestCreateHandler_BadRequest(t *testing.T) {
	mockInput := models.BucketCreateUpdateBody{Name: "test-bucket"}

	mockCreate := new(tests.MockCreateFunc[models.BucketCreateUpdateBody, models.Bucket])
	mockCreate.On(
		"Create",
		mock.AnythingOfType("*zap.Logger"),
		mock.Anything,
		uuid.UUIDs(nil),
		mockInput,
	).Return(models.Bucket{}, errors.New("INVALID_DATA"))

	req := httptest.NewRequest(http.MethodPost, "/buckets", nil)
	recorder := httptest.NewRecorder()

	logger := zap.NewNop()
	ctx := context.WithValue(req.Context(), m.LoggerKey, logger)
	claims := models.UserClaims{UserID: uuid.New()}
	ctx = context.WithValue(ctx, models.UserClaimKey{}, claims)
	ctx = context.WithValue(ctx, m.BodyKey{}, mockInput)
	req = req.WithContext(ctx)

	handler := CreateHandler(mockCreate.Create)
	handler(recorder, req)

	mockCreate.AssertExpectations(t)
	expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_DATA"}}
	tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
}

func TestCreateHandler_InvalidUUID(t *testing.T) {
	invalidUUID := "invalid-uuid"

	mockCreate := new(tests.MockCreateFunc[models.BucketCreateUpdateBody, models.Bucket])

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/buckets/%s", invalidUUID), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", invalidUUID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	req = req.WithContext(ctx)

	handler := CreateHandler(mockCreate.Create)
	handler(recorder, req)

	expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_UUID"}}
	tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
}

func TestCreateHandler_BodyExtractionFailure(t *testing.T) {
	mockCreate := new(tests.MockCreateFunc[models.BucketCreateUpdateBody, models.Bucket])

	req := httptest.NewRequest(http.MethodPost, "/buckets", nil)
	recorder := httptest.NewRecorder()

	logger := zap.NewNop()
	ctx := context.WithValue(req.Context(), m.LoggerKey, logger)
	claims := models.UserClaims{UserID: uuid.New()}
	ctx = context.WithValue(ctx, models.UserClaimKey{}, claims)
	req = req.WithContext(ctx)

	handler := CreateHandler(mockCreate.Create)
	handler(recorder, req)

	expected := models.Error{Status: http.StatusInternalServerError, Error: []string{"INTERNAL_SERVER_ERROR"}}
	tests.AssertJSONResponse(t, recorder, http.StatusInternalServerError, expected)
}

func TestGetListHandler(t *testing.T) {
	records := []models.Bucket{
		{
			ID:        uuid.New(),
			Name:      "bucket1",
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Name:      "bucket2",
			CreatedAt: time.Now(),
		},
	}

	mockGetList := new(tests.MockGetListFunc[models.Bucket])
	mockGetList.On(
		"GetList",
		mock.AnythingOfType("*zap.Logger"),
		mock.Anything,
		uuid.UUIDs(nil),
	).Return(records)

	req := httptest.NewRequest(http.MethodGet, "/buckets", nil)
	recorder := httptest.NewRecorder()

	logger := zap.NewNop()
	ctx := context.WithValue(req.Context(), m.LoggerKey, logger)
	claims := models.UserClaims{UserID: uuid.New()}
	ctx = context.WithValue(ctx, models.UserClaimKey{}, claims)
	req = req.WithContext(ctx)

	handler := GetListHandler(mockGetList.GetList)
	handler(recorder, req)

	mockGetList.AssertExpectations(t)
	page := models.Page[models.Bucket]{Data: records}
	tests.AssertJSONResponse(t, recorder, http.StatusOK, page)
}

func TestGetListHandler_InvalidUUID(t *testing.T) {
	invalidUUID := "invalid-uuid"

	mockGetList := new(tests.MockGetListFunc[models.Bucket])

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/buckets/%s/files", invalidUUID), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", invalidUUID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	req = req.WithContext(ctx)

	handler := GetListHandler(mockGetList.GetList)
	handler(recorder, req)

	expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_UUID"}}
	tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
}

func TestGetOneHandler(t *testing.T) {
	testUUID := uuid.New()
	expectedRecord := models.Bucket{
		ID:        testUUID,
		Name:      "test-bucket",
		CreatedAt: time.Now(),
	}

	mockGetOne := new(tests.MockGetOneFunc[models.Bucket])
	mockGetOne.On(
		"GetOne",
		mock.AnythingOfType("*zap.Logger"),
		mock.Anything,
		uuid.UUIDs{testUUID},
	).Return(expectedRecord, nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	claims := models.UserClaims{UserID: uuid.New()}
	ctx = context.WithValue(ctx, models.UserClaimKey{}, claims)
	req = req.WithContext(ctx)

	handler := GetOneHandler(mockGetOne.GetOne)
	handler(recorder, req)

	mockGetOne.AssertExpectations(t)
	tests.AssertJSONResponse(t, recorder, http.StatusOK, expectedRecord)
}

func TestGetOneHandler_NotFound(t *testing.T) {
	testUUID := uuid.New()

	mockGetOne := new(tests.MockGetOneFunc[models.Bucket])
	mockGetOne.On(
		"GetOne",
		mock.AnythingOfType("*zap.Logger"),
		mock.Anything,
		uuid.UUIDs{testUUID},
	).Return(models.Bucket{}, errors.New("RECORD_NOT_FOUND"))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	claims := models.UserClaims{UserID: uuid.New()}
	ctx = context.WithValue(ctx, models.UserClaimKey{}, claims)
	req = req.WithContext(ctx)

	handler := GetOneHandler(mockGetOne.GetOne)
	handler(recorder, req)

	mockGetOne.AssertExpectations(t)
	expected := models.Error{Status: http.StatusNotFound, Error: []string{"RECORD_NOT_FOUND"}}
	tests.AssertJSONResponse(t, recorder, http.StatusNotFound, expected)
}

func TestGetOneHandler_InvalidUUID(t *testing.T) {
	invalidUUID := "invalid-uuid"

	mockGetOne := new(tests.MockGetOneFunc[models.Bucket])

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/buckets/%s", invalidUUID), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", invalidUUID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	req = req.WithContext(ctx)

	handler := GetOneHandler(mockGetOne.GetOne)
	handler(recorder, req)

	expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_UUID"}}
	tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
}

func TestBodyHandler(t *testing.T) {
	testUUID := uuid.New()
	mockInput := models.BucketCreateUpdateBody{Name: "updated-bucket"}

	mockUpdate := new(tests.MockUpdateFunc[models.BucketCreateUpdateBody])
	mockUpdate.On(
		"Update",
		mock.AnythingOfType("*zap.Logger"),
		mock.Anything,
		uuid.UUIDs{testUUID},
		mockInput,
	).Return(nil)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	claims := models.UserClaims{UserID: uuid.New()}
	ctx = context.WithValue(ctx, models.UserClaimKey{}, claims)
	ctx = context.WithValue(ctx, m.BodyKey{}, mockInput)
	req = req.WithContext(ctx)

	handler := BodyHandler(mockUpdate.Update)
	handler(recorder, req)

	mockUpdate.AssertExpectations(t)
	tests.AssertJSONResponse(t, recorder, http.StatusNoContent, nil)
}

func TestBodyHandler_InvalidUUID(t *testing.T) {
	invalidUUID := "invalid-uuid"

	mockUpdate := new(tests.MockUpdateFunc[models.BucketCreateUpdateBody])

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/buckets/%s", invalidUUID), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", invalidUUID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	req = req.WithContext(ctx)

	handler := BodyHandler(mockUpdate.Update)
	handler(recorder, req)

	expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_UUID"}}
	tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
}

func TestBodyHandler_NotFoundWithAPIError(t *testing.T) {
	testUUID := uuid.New()
	mockInput := models.BucketCreateUpdateBody{Name: "updated-bucket"}

	mockUpdate := new(tests.MockUpdateFunc[models.BucketCreateUpdateBody])
	mockUpdate.On(
		"Update",
		mock.AnythingOfType("*zap.Logger"),
		mock.Anything,
		uuid.UUIDs{testUUID},
		mockInput,
	).Return(apierrors.NewAPIError(http.StatusNotFound, "NOT_FOUND"))

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	claims := models.UserClaims{UserID: uuid.New()}
	ctx = context.WithValue(ctx, models.UserClaimKey{}, claims)
	ctx = context.WithValue(ctx, m.BodyKey{}, mockInput)
	req = req.WithContext(ctx)

	handler := BodyHandler(mockUpdate.Update)
	handler(recorder, req)

	mockUpdate.AssertExpectations(t)
	expected := models.Error{Status: http.StatusNotFound, Error: []string{"NOT_FOUND"}}
	tests.AssertJSONResponse(t, recorder, http.StatusNotFound, expected)
}

func TestBodyHandler_GenericError(t *testing.T) {
	testUUID := uuid.New()
	mockInput := models.BucketCreateUpdateBody{Name: "updated-bucket"}

	mockUpdate := new(tests.MockUpdateFunc[models.BucketCreateUpdateBody])
	mockUpdate.On(
		"Update",
		mock.AnythingOfType("*zap.Logger"),
		mock.Anything,
		uuid.UUIDs{testUUID},
		mockInput,
	).Return(errors.New("GENERIC_ERROR"))

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	claims := models.UserClaims{UserID: uuid.New()}
	ctx = context.WithValue(ctx, models.UserClaimKey{}, claims)
	ctx = context.WithValue(ctx, m.BodyKey{}, mockInput)
	req = req.WithContext(ctx)

	handler := BodyHandler(mockUpdate.Update)
	handler(recorder, req)

	mockUpdate.AssertExpectations(t)
	expected := models.Error{Status: http.StatusBadRequest, Error: []string{"GENERIC_ERROR"}}
	tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
}

func TestBodyHandler_BodyExtractionFailure(t *testing.T) {
	testUUID := uuid.New()

	mockUpdate := new(tests.MockUpdateFunc[models.BucketCreateUpdateBody])

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	claims := models.UserClaims{UserID: uuid.New()}
	ctx = context.WithValue(ctx, models.UserClaimKey{}, claims)
	req = req.WithContext(ctx)

	handler := BodyHandler(mockUpdate.Update)
	handler(recorder, req)

	expected := models.Error{Status: http.StatusInternalServerError, Error: []string{"INTERNAL_SERVER_ERROR"}}
	tests.AssertJSONResponse(t, recorder, http.StatusInternalServerError, expected)
}

func TestDeleteHandler(t *testing.T) {
	testUUID := uuid.New()

	mockDelete := new(tests.MockDeleteFunc)
	mockDelete.On(
		"Delete",
		mock.AnythingOfType("*zap.Logger"),
		mock.Anything,
		uuid.UUIDs{testUUID},
	).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	claims := models.UserClaims{UserID: uuid.New()}
	ctx = context.WithValue(ctx, models.UserClaimKey{}, claims)
	req = req.WithContext(ctx)

	handler := DeleteHandler(mockDelete.Delete)
	handler(recorder, req)

	mockDelete.AssertExpectations(t)
	tests.AssertJSONResponse(t, recorder, http.StatusNoContent, nil)
}

func TestDeleteHandler_NotFound(t *testing.T) {
	testUUID := uuid.New()

	mockDelete := new(tests.MockDeleteFunc)
	mockDelete.On(
		"Delete",
		mock.AnythingOfType("*zap.Logger"),
		mock.Anything,
		uuid.UUIDs{testUUID},
	).Return(errors.New("RECORD_NOT_FOUND"))

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/buckets/%s", testUUID.String()), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", testUUID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	claims := models.UserClaims{UserID: uuid.New()}
	ctx = context.WithValue(ctx, models.UserClaimKey{}, claims)
	req = req.WithContext(ctx)

	handler := DeleteHandler(mockDelete.Delete)
	handler(recorder, req)

	mockDelete.AssertExpectations(t)
	expected := models.Error{Status: http.StatusNotFound, Error: []string{"RECORD_NOT_FOUND"}}
	tests.AssertJSONResponse(t, recorder, http.StatusNotFound, expected)
}

func TestDeleteHandler_InvalidUUID(t *testing.T) {
	invalidUUID := "invalid-uuid"

	mockDelete := new(tests.MockDeleteFunc)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/buckets/%s", invalidUUID), nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id0", invalidUUID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)

	logger := zap.NewNop()
	ctx = context.WithValue(ctx, m.LoggerKey, logger)
	req = req.WithContext(ctx)

	handler := DeleteHandler(mockDelete.Delete)
	handler(recorder, req)

	expected := models.Error{Status: http.StatusBadRequest, Error: []string{"INVALID_UUID"}}
	tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, expected)
}
