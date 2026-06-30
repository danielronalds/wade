package web

import (
	"embed"
	"io/fs"
)

//go:generate npm run build
//go:embed dist
var distFiles embed.FS

func Files() (fs.FS, error) {
	return fs.Sub(distFiles, "dist")
}
