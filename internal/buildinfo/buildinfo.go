package buildinfo

import (
	"runtime/debug"
	"sync"
)

var (
	// This value is injected at build time in ./scripts/build.sh.
	version string

	getVersion = sync.OnceValue(func() string {
		// If version is populated we'll happily return that
		if version != "" {
			return version
		}

		// As version hasn't been injected at build time, we'll
		// have to query `ReadBuildInfo` to attempt to get some
		// information about the binary.
		buildInfo, valid := debug.ReadBuildInfo()
		if !valid {
			// If we're unable to read the build info, we'll
			// just return that we do not know the version.
			return "unknown"
		}

		// We'll attempt to find the "vcs.revision" setting
		// as this is still useful information.
		for _, setting := range buildInfo.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}

		// We haven't found anything useful so we'll again just
		// return that we do not know the version.
		return "unknown"
	})
)

func Version() string {
	return getVersion()
}
