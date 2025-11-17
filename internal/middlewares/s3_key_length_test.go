package middlewares

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"api/internal/models"
	"api/internal/tests"

	"github.com/stretchr/testify/assert"
)

func mockFileTransferHandler(w http.ResponseWriter, r *http.Request) {
	data := r.Context().Value(BodyKey{}).(models.FileTransferBody)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(data)
}

func TestS3KeyLengthValidation(t *testing.T) {
	// UUID length is 36 chars
	// Prefix "buckets/" is 8 chars
	// So we have: 8 + 36 + 1 (/) + path + 1 (/) + name = total
	// Max is 1024 bytes

	testCases := []struct {
		name           string
		inputBody      models.FileTransferBody
		expectedStatus int
		expectedErrors []string
	}{
		{
			name: "Valid - short path and name",
			inputBody: models.FileTransferBody{
				Name: "file.txt",
				Path: "/folder",
				Type: "file",
				Size: 100,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Valid - maximum length path and name",
			inputBody: models.FileTransferBody{
				// Total: 8 (buckets/) + 36 (uuid) + 1 (/) + 700 (path) + 1 (/) + 250 (name) + 4 (.txt) = 1000 bytes
				// Name is 254 chars total (under 255 limit)
				Name: strings.Repeat("a", 250) + ".txt",
				Path: "/" + strings.Repeat("b", 700),
				Type: "file",
				Size: 100,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid - exceeds S3 key length limit",
			inputBody: models.FileTransferBody{
				// Total: 8 (buckets/) + 36 (uuid) + 1 (/) + 950 (path) + 1 (/) + 100 (name) = 1096 bytes > 1024
				Name: strings.Repeat("a", 100) + ".txt",
				Path: "/" + strings.Repeat("b", 950),
				Type: "file",
				Size: 100,
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'FileTransferBody.Name' Error:Field validation for 'Name' failed on the 's3keylength' tag",
			},
		},
		{
			name: "Invalid - Unicode characters taking multiple bytes",
			inputBody: models.FileTransferBody{
				// Unicode characters can take 2-4 bytes each
				// Chinese chars typically take 3 bytes in UTF-8
				// Total: 8 + 36 + 1 + (250 * 3 bytes) + 1 + (80 * 3 bytes) = 8 + 36 + 1 + 750 + 1 + 240 + 4 = 1040 bytes
				// Name is 84 chars (under 255 limit), Path is 250 chars (under 1024 limit)
				Name: strings.Repeat("文", 80) + ".txt",
				Path: "/" + strings.Repeat("档", 250),
				Type: "file",
				Size: 100,
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'FileTransferBody.Name' Error:Field validation for 'Name' failed on the 's3keylength' tag",
			},
		},
		{
			name: "Valid - root path",
			inputBody: models.FileTransferBody{
				Name: "file.txt",
				Path: "/",
				Type: "file",
				Size: 100,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Valid - path ending with slash",
			inputBody: models.FileTransferBody{
				Name: "file.txt",
				Path: "/folder/subfolder/",
				Type: "file",
				Size: 100,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Valid - deeply nested path",
			inputBody: models.FileTransferBody{
				Name: "document.pdf",
				Path: "/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t/u/v/w/x/y/z",
				Type: "file",
				Size: 100,
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.inputBody)
			req := httptest.NewRequest(
				http.MethodPost,
				"/test",
				bytes.NewBuffer(bodyBytes),
			)
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			handler := Validate[models.FileTransferBody](http.HandlerFunc(mockFileTransferHandler))
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code)

			if tt.expectedStatus == http.StatusBadRequest {
				errors := models.Error{Status: tt.expectedStatus, Error: tt.expectedErrors}
				tests.AssertJSONResponse(t, recorder, http.StatusBadRequest, errors)
			}
		})
	}
}
