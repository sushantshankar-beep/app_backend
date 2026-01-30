package utils

import (
	"bytes"
	"html/template"
)

func RenderHTML(rawHTML string, data any) (template.HTML, error) {
	tmpl, err := template.New("providerAgreement").Parse(rawHTML)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return template.HTML(buf.String()), nil
}
