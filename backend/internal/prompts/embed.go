package prompts

import (
	"bytes"
	"embed"
	"text/template"
)

//go:embed *.tmpl
var files embed.FS

func Render(name string, data any) (string, error) {
	t, err := template.ParseFS(files, name+".tmpl")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
