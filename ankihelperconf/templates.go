package ankihelperconf

import (
	"encoding/json"
	"github.com/joomcode/errorx"
	"strings"
	"text/template"
)

const (
	templateOpen  = "$$"
	templateClose = "$$"
)

func ParseTextTemplate(name string, text string) (*template.Template, error) {
	return template.New(name).Delims(templateOpen, templateClose).
		Funcs(template.FuncMap{
			"to_json": ToJson,
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
