package helpers

import (
	"api/internal/models"
	"api/internal/tests"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestValidate struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Filename string `json:"filename" validate:"filename"`
	Type     string `json:"type" validate:"omitempty,oneof=file folder"`
}

func mockNextHandler(w http.ResponseWriter, r *http.Request) {
	data := r.Context().Value(BodyKey{}).(TestValidate)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(data)
}

func TestValidateMiddleware(t *testing.T) {
	testCases := []struct {
		name           string
		inputBody      string
		expectedStatus int
		expectedErrors []string
	}{
		{
			name:           "Valid request body",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "filename": "file.txt"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "No filename validation for folders",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "folder1"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid JSON body",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "filename": "file.txt"`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{"failed to decode body"},
		},
		{
			name:           "Missing required fields",
			inputBody:      `{"name": "", "email": "", "filename": "file.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Name' Error:Field validation for 'Name' failed on the 'required' tag",
				"Key: 'TestValidate.Email' Error:Field validation for 'Email' failed on the 'required' tag",
			},
		},
		{
			name:           "Invalid email format",
			inputBody:      `{"name": "John Doe", "email": "invalid-email", "type": "file", "filename": "file.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Email' Error:Field validation for 'Email' failed on the 'email' tag",
			},
		},
		{
			name:           "Invalid filename format",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file with space.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.inputBody))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			handler := Validate[TestValidate](http.HandlerFunc(mockNextHandler))
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code)

			if tt.expectedStatus == http.StatusBadRequest {
				errors := models.Error{Status: tt.expectedStatus, Error: tt.expectedErrors}
				tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, errors)
			}
		})
	}
}
