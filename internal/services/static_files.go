package services

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type StaticFileService struct {
	staticPath      string
	discoveredFiles map[string]string
}

func NewStaticFileService(directory string) (*StaticFileService, error) {
	var staticPath string

	if !filepath.IsAbs(directory) {
		workDir, _ := os.Getwd()
		staticPath = filepath.Join(workDir, directory)
	} else {
		staticPath = directory
	}

	service := &StaticFileService{
		staticPath:      staticPath,
		discoveredFiles: make(map[string]string),
	}

	if err := service.discoverFiles(); err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}
	return service, nil
}

func (s *StaticFileService) discoverFiles() error {
	return s.walkDirectory(s.staticPath, "")
}

// walkDirectory recursively traverses a directory tree to discover static files.
// It maps URL routes to filesystem paths and handles nested directory structures.
//
// Parameters:
//   - dirPath: The filesystem directory path to traverse
//   - urlPrefix: The URL path prefix for files in this directory (empty for root)
//
// Returns:
//   - error: Any error encountered during directory traversal
//
// The function performs the following operations:
//  1. Reads all entries in the current directory
//  2. For subdirectories: recursively calls itself with updated URL prefix
//  3. For files: checks if serveable and maps route path to filesystem path
//  4. Normalizes path separators for cross-platform compatibility
func (s *StaticFileService) walkDirectory(dirPath, urlPrefix string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())

		if entry.IsDir() {
			subUrlPrefix := filepath.Join(urlPrefix, entry.Name())
			if err := s.walkDirectory(fullPath, subUrlPrefix); err != nil {
				zap.L().Warn("failed to walk subdirectory", zap.String("dir", fullPath), zap.Error(err))
				continue
			}
		} else {
			// Add file to discovered files
			routePath := filepath.Join(urlPrefix, entry.Name())
			routePath = "/" + strings.ReplaceAll(routePath, "\\", "/") // Normalize path separators

			if s.isServeableFile(entry.Name()) {
				relativePath := filepath.Join(urlPrefix, entry.Name())
				s.discoveredFiles[routePath] = relativePath
			}
		}
	}
	zap.L().Debug("file discovery completed", zap.String("directory", dirPath), zap.Int("total_files", len(s.discoveredFiles)))
	return nil
}

func (s *StaticFileService) isServeableFile(fileName string) bool {
	// Serve all common static file types
	staticExtensions := []string{
		".ico", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp",
		".json", ".txt", ".xml", ".pdf",
		".html", ".css", ".js", ".map",
		".woff", ".woff2", ".ttf", ".eot",
		".manifest", ".webmanifest",
	}
	for _, ext := range staticExtensions {
		if strings.HasSuffix(fileName, ext) {
			return true
		}
	}
	return false
}

func (s *StaticFileService) Routes() chi.Router {
	r := chi.NewRouter()
	for routePath := range s.discoveredFiles {
		r.Get(routePath, func(w http.ResponseWriter, req *http.Request) {
			s.serveFile(w, req, req.URL.Path)
		})
	}

	// SPA fallback - serve index.html for all other routes not matched above
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		s.serveSPAFallback(w, req)
	})
	return r
}

func (s *StaticFileService) serveFile(w http.ResponseWriter, r *http.Request, requestPath string) {
	// Check if file was discovered at startup (whitelist approach)
	relativePath, exists := s.discoveredFiles[requestPath]
	if !exists {
		http.NotFound(w, r)
		return
	}

	fullPath := filepath.Join(s.staticPath, relativePath)

	s.secureServeFile(w, r, fullPath)
}

func (s *StaticFileService) serveSPAFallback(w http.ResponseWriter, r *http.Request) {
	// Get relative path for index.html
	relativePath := s.discoveredFiles["/index.html"]
	fullPath := filepath.Join(s.staticPath, relativePath)
	s.secureServeFile(w, r, fullPath)
}

func (s *StaticFileService) secureServeFile(w http.ResponseWriter, r *http.Request, filePath string) {

	s.setSecurityHeaders(w, filePath)

	http.ServeFile(w, r, filePath)
}

func (s *StaticFileService) setSecurityHeaders(w http.ResponseWriter, filePath string) {

	w.Header().Set("X-Content-Type-Options", "nosniff")

	if strings.HasSuffix(filePath, ".html") {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:")
	}

	w.Header().Set("X-Frame-Options", "DENY")

	w.Header().Set("X-XSS-Protection", "1; mode=block")
}
