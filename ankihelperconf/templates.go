package ankihelperconf

import "text/template"

const (
	templateOpen  = "$$"
	templateClose = "$$"
)

func ParseTextTemplate(name string, text string) (*template.Template, error) {
	return template.New(name).Delims(templateOpen, templateClose).Parse(text)
}
