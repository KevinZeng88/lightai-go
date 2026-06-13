//go:build web

package web

import (
	"embed"
	"io/fs"
)

//go:embed dist
var DistFS embed.FS

// GetDist returns the embedded web/dist filesystem.
func GetDist() (fs.FS, error) {
	return fs.Sub(DistFS, "dist")
}
