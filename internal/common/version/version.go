// Package version provides build-time version information.
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version of the build.
	Version = "0.1.0"

	// GitCommit is the git commit hash at build time.
	GitCommit = "unknown"

	// BuildTime is the ISO 8601 build timestamp.
	BuildTime = "unknown"
)

// Info holds the complete version information.
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Get returns the current version information.
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a single-line version string.
func String() string {
	return fmt.Sprintf("lightai-go %s (commit: %s, built: %s, go: %s, %s/%s)",
		Version, GitCommit, BuildTime, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
