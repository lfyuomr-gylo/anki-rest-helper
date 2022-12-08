package templatex

import (
	"strings"
	"text/template"
)

func Execute(tmpl *template.Template, data any) (string, error) {
	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", err
	}
	return result.String(), nil
}
