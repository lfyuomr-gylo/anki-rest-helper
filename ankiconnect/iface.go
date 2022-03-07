package ankiconnect

type NoteID int64

type FieldUpdate struct {
	// one of
	Value     string
	AudioData []byte
}

type CreateModelParams struct {
	ModelName     string                    `json:"modelName"`
	InOrderFields []string                  `json:"inOrderFields"`
	CSS           string                    `json:"css"`
	IsCloze       bool                      `json:"isCloze"`
	CardTemplates []CreateModelCardTemplate `json:"cardTemplates"`
}

type CreateModelCardTemplate struct {
	Name  string `json:"Name"`
	Front string `json:"Front"`
	Back  string `json:"Back"`
}

type API interface {
	FindNotes(query string) ([]NoteID, error)
	NotesInfo(noteIDs []NoteID) (map[NoteID]NoteInfo, error)
	UpdateNoteFields(noteID NoteID, fields map[string]FieldUpdate) error
	ModelNames() ([]string, error)
	CreateModel(params CreateModelParams) error
}
