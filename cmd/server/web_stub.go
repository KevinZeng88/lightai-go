//go:build !web

package main

import (
	"io/fs"
	"net/http"
)

var webAssetsAvailable bool

func getWebAssets() (fs.FS, error) {
	return nil, fs.ErrNotExist
}

func serveEmbeddedWeb(mux *http.ServeMux) {
	mux.HandleFunc("GET /", serveFallbackPage)
}
