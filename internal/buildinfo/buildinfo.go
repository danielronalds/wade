package buildinfo

import "runtime/debug"

// Version returns the main module version embedded at build time.
func Version() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok || buildInfo.Main.Version == "" {
		return "(devel)"
	}
	return buildInfo.Main.Version
}
