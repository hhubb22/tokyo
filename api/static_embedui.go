//go:build embedui

package api

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist_placeholder/* dist/*
var distFS embed.FS

func staticHandler() http.Handler {
	if dist, err := fs.Sub(distFS, "dist"); err == nil {
		if _, err := fs.Stat(dist, "index.html"); err == nil {
			return http.FileServer(http.FS(dist))
		}
	}

	placeholder, err := fs.Sub(distFS, "dist_placeholder")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(placeholder))
}
