//go:build !web

package web

import (
	"io/fs"
)

// GetDist returns nil when web assets are not embedded.
func GetDist() (fs.FS, error) {
	return nil, fs.ErrNotExist
}
