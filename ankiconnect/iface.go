package ankiconnect

import "io"

type NoteID int64

type CardID int64

type FieldUpdate struct {
	// one of
	Value     *string // what value to write to the field
	AudioData []byte  // make field to contain specified Audio. Any previous content of the field is reset.
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
	FindCards(query string) ([]CardID, error)
	NotesInfo(noteIDs []NoteID) (map[NoteID]NoteInfo, error)
	UpdateNoteFields(noteID NoteID, fields map[string]FieldUpdate) error
	ModelNames() ([]string, error)
	CreateModel(params CreateModelParams) error
	ChangeDeck(deckName string, noteIDs []CardID) error
	StoreMediaFile(fileName string, fileData io.Reader, replaceExisting bool) error
	AddTags(noteIDs []NoteID, tags []string) error
}
