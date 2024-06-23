package noteprocessing

import "github.com/joomcode/errorx"

// Modification is a command that a script may return for the helper to execute.
type Modification struct {
	// oneof
	SetField        *map[string]string `json:"set_field"`
	SetFieldIfEmpty *map[string]string `json:"set_field_if_empty"`
	AddTag          *string            `json:"add_tag"`
}

func (m Modification) Validate() error {
	fieldsSet := 0
	if m.SetField != nil {
		fieldsSet++
	}
	if m.AddTag != nil {
		fieldsSet++
	}
	if m.SetFieldIfEmpty != nil {
		fieldsSet++
	}

	if fieldsSet != 1 {
		return errorx.IllegalFormat.New("invalid note modification command has %d top-level keys instead of one: %+v", fieldsSet, m)
	}
	return nil
}

// TemplateData is the data that's available in the script configuration template (in the definition of the CLI args).
type TemplateData struct {
	Note NoteData
}

type NoteData struct {
	Fields map[string]string
	Tags   []string
}
