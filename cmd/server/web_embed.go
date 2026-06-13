//go:build web

package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web/dist
var webDistFS embed.FS

func init() {
	webAssetsAvailable = true
}

func getWebAssets() (fs.FS, error) {
	return fs.Sub(webDistFS, "web/dist")
}

func serveEmbeddedWeb(mux *http.ServeMux) {
	distFS, err := getWebAssets()
	if err != nil {
		mux.HandleFunc("GET /", serveFallbackPage)
		return
	}

	fileServer := http.FileServer(http.FS(distFS))
	mux.Handle("GET /assets/", fileServer)
	mux.Handle("GET /favicon.ico", fileServer)
	mux.Handle("GET /favicon.png", fileServer)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		data, err := fs.ReadFile(distFS, path)
		if err == nil {
			contentType := "application/octet-stream"
			switch {
			case len(path) > 5 && path[len(path)-5:] == ".html":
				contentType = "text/html; charset=utf-8"
			case len(path) > 4 && path[len(path)-4:] == ".css":
				contentType = "text/css"
			case len(path) > 3 && path[len(path)-3:] == ".js":
				contentType = "application/javascript"
			case len(path) > 4 && path[len(path)-4:] == ".svg":
				contentType = "image/svg+xml"
			case len(path) > 4 && path[len(path)-4:] == ".png":
				contentType = "image/png"
			case len(path) > 4 && path[len(path)-4:] == ".ico":
				contentType = "image/x-icon"
			}
			w.Header().Set("Content-Type", contentType)
			w.Write(data)
			return
		}

		// SPA fallback.
		data, err = fs.ReadFile(distFS, "index.html")
		if err != nil {
			serveFallbackPage(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})
}
