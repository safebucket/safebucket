package middlewares

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests"

	"github.com/stretchr/testify/assert"
)

type TestValidate struct {
	Name       string `json:"name"       validate:"required"`
	Email      string `json:"email"      validate:"required,email"`
	Filename   string `json:"filename"   validate:"filename"`
	Foldername string `json:"foldername" validate:"omitempty,foldername"`
	Type       string `json:"type"       validate:"omitempty,oneof=file folder"`
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
			name:           "Valid filename with spaces",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "my file.txt"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid filename with special characters",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "a&b+c#d'e,f.txt"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid filename without extension",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "Makefile"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid dotfile",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": ".gitignore"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid filename with unicode",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "café 日本語.pdf"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid JSON body",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "filename": "file.txt"`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{"BAD_REQUEST"},
		},
		{
			name:           "Missing required fields",
			inputBody:      `{"name": "", "email": "", "filename": "file.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{"FIELD_REQUIRED", "FIELD_REQUIRED"},
		},
		{
			name:           "Invalid email format",
			inputBody:      `{"name": "John Doe", "email": "invalid-email", "type": "file", "filename": "file.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{"INVALID_EMAIL"},
		},
		{
			name:           "Invalid filename - prohibited characters",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file/path.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{"INVALID_FILENAME"},
		},
		{
			name:           "Invalid filename - windows reserved name",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "CON.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{"INVALID_FILENAME"},
		},
		{
			name:           "Invalid filename - trailing space",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt "}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{"INVALID_FILENAME"},
		},
		{
			name:           "Invalid filename - trailing dot",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file."}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{"INVALID_FILENAME"},
		},
		{
			name:           "Invalid filename - rtl override spoof",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "Invoice` + "\u202e" + `fdp.exe"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{"INVALID_FILENAME"},
		},
		{
			name:           "Valid filename with extra allowed symbols",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "@100%$ data{v2}~=;!^.txt"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid filename - disallowed ascii symbol",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "report|home.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{"INVALID_FILENAME"},
		},
		{
			name:           "Valid foldername with dots",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "file.txt", "foldername": "humanlog_0.7.8_linux_amd64"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid foldername with parentheses and brackets",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "file.txt", "foldername": "backup (1) [old]"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid foldername with special characters",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "file.txt", "foldername": "a&b+c#d'e,f"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid foldername - prohibited characters",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "file.txt", "foldername": "sub/folder"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{"INVALID_FILENAME"},
		},
		{
			name:           "Invalid foldername - dot only",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "file.txt", "foldername": ".."}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{"INVALID_FILENAME"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/test",
				bytes.NewBufferString(tt.inputBody),
			)
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
