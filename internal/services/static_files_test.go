package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticFileService_ServesJavaScriptModules(t *testing.T) {
	service, err := NewStaticFileService(
		fstest.MapFS{
			"index.html":                     &fstest.MapFile{Data: []byte("<!doctype html>")},
			"assets/pdf.worker.min-test.mjs": &fstest.MapFile{Data: []byte("export {};")},
		},
		"",
		"",
		false,
	)
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/assets/pdf.worker.min-test.mjs", nil)
	response := httptest.NewRecorder()

	service.Routes().ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), "export {};")
	assert.True(t, strings.HasPrefix(response.Header().Get("Content-Type"), "text/javascript"))
}
