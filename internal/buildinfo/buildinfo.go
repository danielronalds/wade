package buildinfo

import "runtime/debug"

var releaseVersion string

// Version returns the main module version embedded at build time.
func Version() string {
	if releaseVersion != "" {
		return releaseVersion
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok || buildInfo.Main.Version == "" {
		return "(devel)"
	}
	return buildInfo.Main.Version
}
