package ankiconnectmock

import (
	"anki-rest-enhancer/ankiconnect"
	"github.com/joomcode/errorx"
)

type API struct {
	FindNotesFunc        func(query string) ([]ankiconnect.NoteID, error)
	NotesInfoFunc        func(noteIDs []ankiconnect.NoteID) (map[ankiconnect.NoteID]ankiconnect.NoteInfo, error)
	UpdateNoteFieldsFunc func(noteID ankiconnect.NoteID, fields map[string]ankiconnect.FieldUpdate) error
	ModelNamesFunc       func() ([]string, error)
	CreateModelFunc      func(params ankiconnect.CreateModelParams) error
	FindCardsFunc        func(query string) ([]ankiconnect.CardID, error)
	ChangeDeckFunc       func(deckName string, noteIDs []ankiconnect.CardID) error
}

var _ ankiconnect.API = (*API)(nil)

func (api *API) Reset() {
	*api = API{}
}

func (api *API) FindNotes(query string) ([]ankiconnect.NoteID, error) {
	if behaviour := api.FindNotesFunc; behaviour != nil {
		return behaviour(query)
	}
	panic(errorx.Panic(errorx.NotImplemented.New("Mock behaviour is not set for method FindNotes")))
}

func (api *API) NotesInfo(noteIDs []ankiconnect.NoteID) (map[ankiconnect.NoteID]ankiconnect.NoteInfo, error) {
	if behaviour := api.NotesInfoFunc; behaviour != nil {
		return behaviour(noteIDs)
	}
	panic(errorx.Panic(errorx.NotImplemented.New("Mock behaviour is not set for method NotesInfo")))
}

func (api *API) UpdateNoteFields(noteID ankiconnect.NoteID, fields map[string]ankiconnect.FieldUpdate) error {
	if behaviour := api.UpdateNoteFieldsFunc; behaviour != nil {
		return behaviour(noteID, fields)
	}
	panic(errorx.Panic(errorx.NotImplemented.New("Mock behaviour is not set for method UpdateNoteFields")))
}

func (api *API) ModelNames() ([]string, error) {
	if behaviour := api.ModelNamesFunc; behaviour != nil {
		return behaviour()
	}
	panic(errorx.Panic(errorx.NotImplemented.New("Mock behaviour is not set for method ModelNames")))
}

func (api *API) CreateModel(params ankiconnect.CreateModelParams) error {
	if behaviour := api.CreateModelFunc; behaviour != nil {
		return behaviour(params)
	}
	panic(errorx.Panic(errorx.NotImplemented.New("Mock behaviour is not set for method CreateModel")))
}

func (api *API) FindCards(query string) ([]ankiconnect.CardID, error) {
	if behaviour := api.FindCardsFunc; behaviour != nil {
		return behaviour(query)
	}
	panic(errorx.Panic(errorx.NotImplemented.New("Mock behaviour is not set for method FindCards")))
}

func (api *API) ChangeDeck(deckName string, noteIDs []ankiconnect.CardID) error {
	if behaviour := api.ChangeDeckFunc; behaviour != nil {
		return behaviour(deckName, noteIDs)
	}
	panic(errorx.Panic(errorx.NotImplemented.New("Mock behaviour is not set for method ChangeDeck")))
}
