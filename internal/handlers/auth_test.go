package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api/internal/tests"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOpenIDBeginHandler(t *testing.T) {
	mockOpenIDBegin := new(tests.MockOpenIDBeginFunc)
	providerName := "google"
	redirectURL := "https://oauth.google.com/auth"

	mockOpenIDBegin.On(
		"OpenIDBegin",
		providerName,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return(redirectURL, nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", providerName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := OpenIDBeginHandler(mockOpenIDBegin.OpenIDBegin)
	handler(recorder, req)

	mockOpenIDBegin.AssertExpectations(t)
	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, redirectURL, recorder.Header().Get("Location"))

	args := mockOpenIDBegin.Calls[0].Arguments
	state := args.Get(1)
	nonce := args.Get(2)

	cookies := recorder.Result().Cookies()
	assert.Len(t, cookies, 2)
	assert.Equal(t, "state", cookies[0].Name)
	assert.Equal(t, state, cookies[0].Value)
	assert.Equal(t, "nonce", cookies[1].Name)
	assert.Equal(t, nonce, cookies[1].Value)
}

func TestOpenIDBeginHandler_ProviderNotFound(t *testing.T) {
	mockOpenIDBegin := new(tests.MockOpenIDBeginFunc)
	providerName := "github"

	mockOpenIDBegin.On(
		"OpenIDBegin",
		providerName,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return("", errors.New("PROVIDER_NOT_FOUND"))

	req := httptest.NewRequest(http.MethodGet, "/auth/github", nil)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", providerName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := OpenIDBeginHandler(mockOpenIDBegin.OpenIDBegin)
	handler(recorder, req)

	mockOpenIDBegin.AssertExpectations(t)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "PROVIDER_NOT_FOUND")

	cookies := recorder.Result().Cookies()
	assert.Empty(t, cookies)
}

func TestOpenIDCallbackHandler(t *testing.T) {
	mockOpenIDCallback := new(tests.MockOpenIDCallbackFunc)
	providerName := "google"
	webURL := "https://safebucket.com"
	code := "test_code"
	state := "test_state"
	nonce := "test_nonce"
	accessToken := "test_access_token"
	refreshToken := "test_refresh_token"

	mockOpenIDCallback.On(
		"OpenIDCallback",
		mock.Anything, // ctx context.Context
		mock.Anything, // logger *zap.Logger
		providerName,
		code,
		nonce,
	).Return(accessToken, refreshToken, nil)

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/auth/callback/%s?code=%s&state=%s", providerName, code, state),
		nil,
	)
	recorder := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", providerName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	req.AddCookie(&http.Cookie{Name: "state", Value: state})
	req.AddCookie(&http.Cookie{Name: "nonce", Value: nonce})

	handler := OpenIDCallbackHandler(webURL, mockOpenIDCallback.OpenIDCallback)
	handler(recorder, req)

	mockOpenIDCallback.AssertExpectations(t)

	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, fmt.Sprintf("%s/auth/complete", webURL), recorder.Header().Get("Location"))

	cookies := recorder.Result().Cookies()
	assert.Len(t, cookies, 3)

	expectedCookies := map[string]string{
		"safebucket_access_token":  accessToken,
		"safebucket_auth_provider": providerName,
		"safebucket_refresh_token": refreshToken,
	}

	for _, cookie := range cookies {
		expectedValue, exists := expectedCookies[cookie.Name]
		assert.True(t, exists, "Unexpected cookie: %s", cookie.Name)
		assert.Equal(t, expectedValue, cookie.Value)
		assert.Equal(t, "/", cookie.Path)
		assert.True(t, cookie.Expires.After(time.Now().Add(364*24*time.Hour)))
	}
}

func TestOpenIDCallbackHandler_Errors(t *testing.T) {
	mockOpenIDCallback := new(tests.MockOpenIDCallbackFunc)
	providerName := "google"
	webURL := "https://example.com"
	code := "test_code"
	state := "test_state"
	nonce := "test_nonce"

	testCases := []struct {
		name               string
		setupRequest       func(*http.Request)
		expectedStatusCode int
		expectedError      string
	}{
		{
			name: "Missing state cookie",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "nonce", Value: nonce})
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "state not found",
		},
		{
			name: "State mismatch",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "state", Value: "wrong_state"})
				req.AddCookie(&http.Cookie{Name: "nonce", Value: nonce})
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "state does not match",
		},
		{
			name: "Missing nonce cookie",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "state", Value: state})
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedError:      "nonce not found",
		},
		{
			name: "OpenIDCallback error",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "state", Value: state})
				req.AddCookie(&http.Cookie{Name: "nonce", Value: nonce})
				mockOpenIDCallback.On(
					"OpenIDCallback",
					mock.Anything, // ctx context.Context
					mock.Anything, // logger *zap.Logger
					providerName,
					code,
					nonce,
				).Return("", "", errors.New("OpenIDCallback error"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedError:      "OpenIDCallback error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/auth/callback/%s?code=%s&state=%s", providerName, code, state),
				nil,
			)
			recorder := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("provider", providerName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			tc.setupRequest(req)

			handler := OpenIDCallbackHandler(webURL, mockOpenIDCallback.OpenIDCallback)
			handler(recorder, req)

			assert.Equal(t, tc.expectedStatusCode, recorder.Code)
			assert.Contains(t, recorder.Body.String(), tc.expectedError)
		})
	}

	mockOpenIDCallback.AssertExpectations(t)
}
