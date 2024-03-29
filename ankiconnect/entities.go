package ankiconnect

import (
	"encoding/json"
	"reflect"
)

type action string

var actionParamsMapping = map[reflect.Type]action{}
var actionResultMapping = map[action]reflect.Type{}

func declareAction(name string, paramsProto interface{}, resultProto interface{}) action {
	if reflect.ValueOf(paramsProto).Kind() == reflect.Ptr {
		paramsProto = reflect.ValueOf(paramsProto).Elem().Interface()
	}
	actionParamsMapping[reflect.TypeOf(paramsProto)] = action(name)

	if reflect.ValueOf(resultProto).Kind() == reflect.Ptr {
		resultProto = reflect.ValueOf(resultProto).Elem().Interface()
	}
	actionResultMapping[action(name)] = reflect.TypeOf(resultProto)

	return action(name)
}

type requestPayload struct {
	Action  action      `json:"action"`
	Version int         `json:"version"`
	Params  interface{} `json:"params,omitempty"`
}

type responsePayload struct {
	Error  *string         `json:"error"`
	Result json.RawMessage `json:"result"`
}

//goland:noinspection GoUnusedGlobalVariable
var actionFindNotes = declareAction("findNotes", findNotesParams{}, findNotesResult{})

type findNotesParams struct {
	Query string `json:"query"`
}

type findNotesResult []NoteID

//goland:noinspection GoUnusedGlobalVariable
var actionFindCards = declareAction("findCards", findCardsParams{}, findCardsResult{})

type findCardsParams struct {
	Query string `json:"query"`
}

type findCardsResult []CardID

//goland:noinspection GoUnusedGlobalVariable
var actionNotesInfo = declareAction("notesInfo", notesInfoParams{}, notesInfoResult{})

type notesInfoParams struct {
	NoteIDs []NoteID `json:"notes"`
}

type notesInfoResult []noteInfo

type noteInfo struct {
	NoteID    NoteID                `json:"noteId"`
	ModelName string                `json:"modelName"`
	Tags      []string              `json:"tags"`
	Fields    map[string]fieldValue `json:"fields"`
}

type fieldValue struct {
	Order int    `json:"order"`
	Value string `json:"value"`
}

//goland:noinspection GoUnusedGlobalVariable
var actionUpdateNoteFields = declareAction("updateNoteFields", updateNoteFieldsParams{}, updateNoteFieldsResult{})

type updateNoteFieldsParams struct {
	Note updateNoteFieldsNote `json:"note"`
}

type updateNoteFieldsNote struct {
	ID     NoteID                  `json:"id"`
	Fields map[string]string       `json:"fields"`
	Audio  []updateNoteFieldsAudio `json:"audio"`
}

type updateNoteFieldsAudio struct {
	FileName   string   `json:"filename"`
	Base64Data string   `json:"data"`
	Fields     []string `json:"fields"`
}

type updateNoteFieldsResult struct {
	// nop
}

//goland:noinspection GoUnusedGlobalVariable
var actionModelNames = declareAction("modelNames", modelNamesParams{}, modelNamesResult{})

type modelNamesParams struct {
	// nop
}

type modelNamesResult []string

//goland:noinspection GoUnusedGlobalVariable
var actionCreateModel = declareAction("createModel", createModelParams{}, createModelResult{})

// Right now AnkiConnect model creation request structure perfectly fits the API needs,
// so it is reused both as API parameter and as a DTO.
type createModelParams = CreateModelParams

type createModelResult struct {
	// nop -- we don't need createModel action result at the time, so don't do any unmarshalling here
}

//goland:noinspection GoUnusedGlobalVariable
var actionChangeDeck = declareAction("changeDeck", changeDeckParams{}, changeDeckResult{})

type changeDeckParams struct {
	Deck  string   `json:"deck"`
	Cards []CardID `json:"cards"`
}

type changeDeckResult struct {
	// nop
}

//goland:noinspection GoUnusedGlobalVariable
var actionStoreMediaFile = declareAction("storeMediaFile", storeMediaFileParams{}, storeMediaFileResult(""))

type storeMediaFileParams struct {
	FileName string `json:"filename"`
	// DeleteExisting specifies whether to overwrite existing file with the same name.
	// NOTE: do not use omitempty for this field as AnkiConnect treats absence of this parameter as 'true'!
	DeleteExisting bool `json:"deleteExisting"`

	// oneof:
	DataBase64 string `json:"data,omitempty"`
	FilePath   string `json:"path,omitempty"`
	URL        string `json:"url,omitempty"`
}

type storeMediaFileResult string

//goland:noinspection GoUnusedGlobalVariable
var actionAddTags = declareAction("addTags", addTagsParams{}, addTagsResult{})

type addTagsParams struct {
	Notes []NoteID `json:"notes"`
	// Tags is a space-separated list of tags, as per
	// https://github.com/FooSoft/anki-connect/blob/42a6de39b317a9471e4ad430e13e0f6a5d0d5d58/plugin/__init__.py#L799
	Tags string `json:"tags"`
}

type addTagsResult struct {
	// nop
}
