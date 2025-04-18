package buildinfo

import ()

var (
	// This value is injected at build time in ./scripts/build.sh.
	version string
)

func Version() string {
	return version
}
