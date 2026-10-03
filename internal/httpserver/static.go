package httpserver

import (
	"mime"
	"os"
	"path/filepath"
	"strings"
)

func staticFilePath(requestPath string) (string, bool) {
	const prefix = "/static/"

	if !strings.HasPrefix(requestPath, prefix) {
		return "", false
	}

	relativePath := strings.TrimPrefix(requestPath, prefix)
	if relativePath == "" {
		return "", false
	}

	staticRoot, err := filepath.Abs("public/static")
	if err != nil {
		return "", false
	}

	candidate := filepath.Join(staticRoot, filepath.FromSlash(relativePath))

	relativeToRoot, err := filepath.Rel(staticRoot, candidate)
	if err != nil {
		return "", false
	}

	if relativeToRoot == ".." ||
		strings.HasPrefix(relativeToRoot, ".."+string(filepath.Separator)) {
		return "", false
	}

	return candidate, true
}

func serveStaticFile(requestPath string) Response {
	filePath, ok := staticFilePath(requestPath)
	if !ok {
		return notFoundResponse()
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return notFoundResponse()
	}

	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return Response{
		StatusCode: 200,
		StatusText: "OK",
		Headers: map[string]string{
			"Content-Type": contentType,
		},
		Body: data,
	}
}
