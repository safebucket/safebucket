package helpers

import (
	"strings"
)

var previewMimeByExt = map[string]string{
	"png":  "image/png",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"gif":  "image/gif",
	"webp": "image/webp",
	"svg":  "image/svg+xml",
	"bmp":  "image/bmp",
	"ico":  "image/x-icon",
	"avif": "image/avif",

	"mp4":  "video/mp4",
	"webm": "video/webm",
	"ogv":  "video/ogg",
	"mov":  "video/quicktime",
	"m4v":  "video/x-m4v",

	"mp3":  "audio/mpeg",
	"wav":  "audio/wav",
	"ogg":  "audio/ogg",
	"m4a":  "audio/mp4",
	"flac": "audio/flac",
	"aac":  "audio/aac",

	"pdf": "application/pdf",

	"json": "application/json",
	"xml":  "application/xml",
	"html": "text/html",
	"htm":  "text/html",
	"css":  "text/css",
	"csv":  "text/csv",
	"tsv":  "text/tab-separated-values",
	"md":   "text/markdown",
}

func PreviewMimeFromExtension(extension string) string {
	ext := strings.ToLower(strings.TrimPrefix(extension, "."))
	if mime, ok := previewMimeByExt[ext]; ok {
		return mime
	}
	return "text/plain"
}
