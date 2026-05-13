package services

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type StaticFileService struct {
	fsys                       fs.FS
	discoveredFiles            map[string]bool
	configJSON                 []byte
	apiURL                     string
	storageExternalURL         string
	requiresUploadConfirmation bool
}

type ConfigJSON struct {
	APIURL                     string `json:"apiUrl"`
	Environment                string `json:"environment"`
	RequiresUploadConfirmation bool   `json:"requiresUploadConfirmation"`
}

func NewStaticFileService(
	fsys fs.FS,
	apiURL string,
	storageExternalURL string,
	requiresUploadConfirmation bool,
) (*StaticFileService, error) {
	service := &StaticFileService{
		fsys:                       fsys,
		discoveredFiles:            make(map[string]bool),
		apiURL:                     apiURL,
		storageExternalURL:         storageExternalURL,
		requiresUploadConfirmation: requiresUploadConfirmation,
	}

	configData, err := service.buildConfigJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to build config JSON: %w", err)
	}
	service.configJSON = configData

	if err = service.discoverFiles(); err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}

	if _, err = fs.Stat(fsys, "index.html"); err != nil {
		return nil, fmt.Errorf("index.html not found in static files: %w", err)
	}

	return service, nil
}

func (s *StaticFileService) buildConfigJSON() ([]byte, error) {
	config := ConfigJSON{
		APIURL:                     s.apiURL,
		Environment:                "production",
		RequiresUploadConfirmation: s.requiresUploadConfirmation,
	}

	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config JSON: %w", err)
	}

	return data, nil
}

func (s *StaticFileService) serveConfigJSON(w http.ResponseWriter, _ *http.Request) {
	s.setSecurityHeaders(w, "config.json")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	_, err := w.Write(s.configJSON)
	if err != nil {
		zap.L().Error("failed to write response", zap.Error(err))
	}
}

func (s *StaticFileService) discoverFiles() error {
	err := fs.WalkDir(s.fsys, ".", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			zap.L().Warn("error accessing path during file discovery", zap.String("path", filePath), zap.Error(err))
			return nil
		}

		if !d.IsDir() && s.isServableFile(d.Name()) {
			routePath := "/" + filePath
			s.discoveredFiles[routePath] = true
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to discover files: %w", err)
	}

	zap.L().
		Debug("file discovery completed", zap.Int("total_files", len(s.discoveredFiles)))
	return nil
}

func (s *StaticFileService) isServableFile(fileName string) bool {
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

	r.Get("/config.json", s.serveConfigJSON)

	for routePath := range s.discoveredFiles {
		if routePath == "/config.json" {
			continue
		}
		r.Get(routePath, func(w http.ResponseWriter, req *http.Request) {
			s.serveFile(w, req, req.URL.Path)
		})
	}

	// SPA fallback - serve index.html for all other routes not matched above.
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		s.serveSPAFallback(w, req)
	})
	return r
}

func (s *StaticFileService) serveFile(w http.ResponseWriter, r *http.Request, requestPath string) {
	if !s.discoveredFiles[requestPath] {
		http.NotFound(w, r)
		return
	}

	fsPath := strings.TrimPrefix(requestPath, "/")
	s.secureServeFile(w, r, fsPath)
}

func (s *StaticFileService) serveSPAFallback(w http.ResponseWriter, r *http.Request) {
	s.secureServeFile(w, r, "index.html")
}

func (s *StaticFileService) secureServeFile(
	w http.ResponseWriter,
	r *http.Request,
	filePath string,
) {
	s.setSecurityHeaders(w, filePath)

	http.ServeFileFS(w, r, s.fsys, filePath)
}

func (s *StaticFileService) setSecurityHeaders(w http.ResponseWriter, filePath string) {
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if strings.HasSuffix(filePath, ".html") {
		connectSrc := fmt.Sprintf("'self' %s", s.storageExternalURL)

		csp := fmt.Sprintf(
			"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src %s",
			connectSrc,
		)
		w.Header().Set("Content-Security-Policy", csp)
	}

	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
}
