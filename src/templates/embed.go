// SPDX-License-Identifier: AGPL-3.0-or-later

package templates

import (
	"embed"
	"io"

	"github.com/tyler-sommer/stick"
)

//go:embed *.twig
var EmbeddedFS embed.FS

// EmbeddedTemplateLoader implements stick.Loader
type EmbeddedTemplateLoader struct {
	FS embed.FS
}

// Load attempts to load the given file
func (l *EmbeddedTemplateLoader) Load(name string) (stick.Template, error) {
	f, err := l.FS.Open(name)
	if err != nil {
		return nil, err
	}
	return &embeddedFileTemplate{name: name, reader: f}, nil
}

type embeddedFileTemplate struct {
	name   string
	reader io.Reader
}

func (t *embeddedFileTemplate) Name() string {
	return t.name
}

func (t *embeddedFileTemplate) Contents() io.Reader {
	return t.reader
}
