package ankihelperconf

import (
	"encoding/json"
	"github.com/joomcode/errorx"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	templateOpen  = "$$"
	templateClose = "$$"
)

func ParseTextTemplate(configDir, name, text string) (*template.Template, error) {
	return template.New(name).Delims(templateOpen, templateClose).
		Funcs(template.FuncMap{
			"to_json":      ToJson,
			"resolve_path": func(path string) string { return ResolvePath(configDir, path) },
		}).
		Parse(text)
}

func ToJson(value any) (string, error) {
	var marshalled strings.Builder
	if err := json.NewEncoder(&marshalled).Encode(value); err != nil {
		return "", errorx.IllegalState.Wrap(err, "failed to marshal to JSON value %+v", value)
	}
	return marshalled.String(), nil
}

func ResolvePath(configDir, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(configDir, path)
	}
	return path
}
