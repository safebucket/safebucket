package middlewares

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"api/internal/models"
	"api/internal/tests"

	"github.com/stretchr/testify/assert"
)

type TestValidate struct {
	Name       string `json:"name"       validate:"required"`
	Email      string `json:"email"      validate:"required,email"`
	Filename   string `json:"filename"   validate:"filename"`
	Path       string `json:"path"       validate:"omitempty,filepath"`
	Type       string `json:"type"       validate:"omitempty,oneof=file folder"`
	BucketName string `json:"bucketname" validate:"omitempty,bucketname"`
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
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid filename with spaces",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "My Document.txt"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid filename with Unicode",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "émile.txt"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid filename with Chinese characters",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "文档.pdf"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid filename with Cyrillic characters",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "файл.txt"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid filename with parentheses",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "report (final).pdf"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid filename with brackets - left bracket",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "photo [2024].jpg"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Valid filename with multiple dots",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.backup.txt"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid extensionless file",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "Makefile"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid hidden file (starts with dot)",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": ".gitignore"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid file ending with dot",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file."}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid folder name",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "My Folder"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid folder with Unicode",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "文件夹"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid folder ending with space",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "folder "}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid folder ending with dot",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "folder."}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid bucket name",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "temp", "bucketname": "My Bucket"}`,
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
			name:           "Invalid filename - path traversal",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "../etc/passwd.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Invalid filename - forward slash",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "path/to/file.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Invalid filename - backslash",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "path\\file.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Invalid filename - percent character",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file%20name.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Invalid filename - special char colon",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file:name.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Invalid filename - special char asterisk",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file*.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Invalid filename - special char question mark",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file?.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Invalid filename - special char quote",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file\".txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Invalid filename - special char pipe",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file|name.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Invalid folder - path traversal",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "../etc"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Invalid folder - percent character",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "folder%name"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Invalid folder - reserved name",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": ".."}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Invalid bucket name - path traversal",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "temp", "bucketname": "../bucket"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.BucketName' Error:Field validation for 'BucketName' failed on the 'bucketname' tag",
			},
		},
		{
			name:           "Invalid bucket name - percent character",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "temp", "bucketname": "bucket%name"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.BucketName' Error:Field validation for 'BucketName' failed on the 'bucketname' tag",
			},
		},
		{
			name:           "Invalid bucket name - special char asterisk",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "folder", "filename": "temp", "bucketname": "bucket*name"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.BucketName' Error:Field validation for 'BucketName' failed on the 'bucketname' tag",
			},
		},
		{
			name:           "Invalid filename - right bracket",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file]name.txt"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Filename' Error:Field validation for 'Filename' failed on the 'filename' tag",
			},
		},
		{
			name:           "Valid path - simple folder",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt", "path": "/folder"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid path - nested folders",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt", "path": "/folder/subfolder/deep"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid path - root",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt", "path": "/"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid path - with Unicode",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt", "path": "/文档/子文件夹"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid path - with spaces and special chars",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt", "path": "/My Folder/Sub-Folder (2024)"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid path - path traversal",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt", "path": "/folder/../etc"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Path' Error:Field validation for 'Path' failed on the 'filepath' tag",
			},
		},
		{
			name:           "Invalid path - backslash",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt", "path": "/folder\\subfolder"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Path' Error:Field validation for 'Path' failed on the 'filepath' tag",
			},
		},
		{
			name:           "Invalid path - percent character",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt", "path": "/folder%20name"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Path' Error:Field validation for 'Path' failed on the 'filepath' tag",
			},
		},
		{
			name:           "Invalid path - brackets",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt", "path": "/folder[2024]"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Path' Error:Field validation for 'Path' failed on the 'filepath' tag",
			},
		},
		{
			name:           "Invalid path - asterisk",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt", "path": "/folder*/subfolder"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Path' Error:Field validation for 'Path' failed on the 'filepath' tag",
			},
		},
		{
			name:           "Invalid path - colon",
			inputBody:      `{"name": "John Doe", "email": "john@example.com", "type": "file", "filename": "file.txt", "path": "/folder:name"}`,
			expectedStatus: http.StatusBadRequest,
			expectedErrors: []string{
				"Key: 'TestValidate.Path' Error:Field validation for 'Path' failed on the 'filepath' tag",
			},
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
