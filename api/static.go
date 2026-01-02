package api

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var distFS embed.FS

func staticHandler() http.Handler {
	dist, _ := fs.Sub(distFS, "dist")
	return http.FileServer(http.FS(dist))
}
