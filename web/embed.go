package web

import (
	"embed"
	"io/fs"
)

//go:generate rm -rf ./dist
//go:generate mkdir -p ./dist/vendor/xterm
//go:generate cp -R ./static/. ./dist/
//go:generate cp ./node_modules/@xterm/xterm/lib/xterm.js ./dist/vendor/xterm/xterm.js
//go:generate cp ./node_modules/@xterm/xterm/lib/xterm.js.map ./dist/vendor/xterm/xterm.js.map
//go:generate cp ./node_modules/@xterm/xterm/css/xterm.css ./dist/vendor/xterm/xterm.css
//go:generate cp ./node_modules/@xterm/xterm/LICENSE ./dist/vendor/xterm/xterm.LICENSE
//go:generate cp ./node_modules/@xterm/addon-fit/lib/addon-fit.js ./dist/vendor/xterm/addon-fit.js
//go:generate cp ./node_modules/@xterm/addon-fit/lib/addon-fit.js.map ./dist/vendor/xterm/addon-fit.js.map
//go:generate cp ./node_modules/@xterm/addon-fit/LICENSE ./dist/vendor/xterm/addon-fit.LICENSE
//go:embed dist
var distFiles embed.FS

func Files() (fs.FS, error) {
	return fs.Sub(distFiles, "dist")
}
