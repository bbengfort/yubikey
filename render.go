package yubikey

import (
	"embed"
	"html/template"
	"io/fs"
	"path/filepath"

	"github.com/gin-gonic/gin/render"
)

// content holds our static web server content.
//
//go:embed templates
//go:embed static
var content embed.FS

type Render struct {
	templates map[string]*template.Template
}

func NewRender(fsys fs.FS, pattern string, includes ...string) (_ *Render, err error) {
	render := &Render{
		templates: make(map[string]*template.Template),
	}

	// Kind of a hack, but parses each top-level *.html file individually and includes
	// the layouts and partials with each template parsed from the file system.
	var names []string
	if names, err = fs.Glob(fsys, pattern); err != nil {
		return nil, err
	}

	for _, name := range names {
		patterns := append([]string{name}, includes...)
		if render.templates[filepath.Base(name)], err = template.ParseFS(fsys, patterns...); err != nil {
			return nil, err
		}
	}
	return render, nil
}

var _ render.HTMLRender = &Render{}

func (r *Render) Instance(name string, data any) render.Render {
	return &render.HTML{
		Template: r.templates[name],
		Name:     name,
		Data:     data,
	}
}
