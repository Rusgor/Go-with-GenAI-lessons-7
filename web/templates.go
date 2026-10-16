// Package web embeds the HTML templates and static assets into the binary.
package web

import (
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates
var templatesFS embed.FS

// pageFiles maps a page name to the page template that fills the layout's "content" block.
var pageFiles = map[string]string{
	"catalog": "templates/catalog.html",
	"login":   "templates/login.html",
}

// Templates parses every page together with the layout and partials from the embedded FS.
// Each page gets its own template set so their "content" blocks do not collide.
func Templates() (map[string]*template.Template, error) {
	pages := make(map[string]*template.Template, len(pageFiles))
	for name, file := range pageFiles {
		t, err := template.ParseFS(templatesFS, "templates/layout.html", "templates/partials/*.html", file)
		if err != nil {
			return nil, fmt.Errorf("parse %s template: %w", name, err)
		}
		pages[name] = t
	}
	return pages, nil
}
