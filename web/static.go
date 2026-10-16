package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var staticFS embed.FS

// StaticFS returns the embedded static assets rooted at "static", so that
// /static/css/style.css maps to css/style.css instead of static/css/style.css.
func StaticFS() (fs.FS, error) {
	return fs.Sub(staticFS, "static")
}
