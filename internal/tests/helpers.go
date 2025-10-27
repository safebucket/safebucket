package tests

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func AssertJSONResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	expectedStatus int,
	expectedPayload interface{},
) {
	t.Helper()

	assert.Equal(t, expectedStatus, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "*", recorder.Header().Get("Access-Control-Allow-Origin"))

	if expectedPayload != nil {
		expectedJSON, err := json.Marshal(expectedPayload)
		assert.NoError(t, err)
		assert.JSONEq(t, string(expectedJSON), recorder.Body.String())
	}
}
