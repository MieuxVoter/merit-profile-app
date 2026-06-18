// SPDX-License-Identifier: AGPL-3.0-or-later

package templates

import (
	"embed"
	"github.com/tyler-sommer/stick"
	"io"
)

//go:embed *.twig
var TemplatesFS embed.FS

type EmbedFSLoader struct {
	FS embed.FS
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

// Load attempts to load the given file
func (l *EmbedFSLoader) Load(name string) (stick.Template, error) {
	f, err := l.FS.Open(name)
	if err != nil {
		return nil, err
	}
	return &embeddedFileTemplate{name: name, reader: f}, nil
}
