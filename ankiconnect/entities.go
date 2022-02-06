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
var actionCreateModel = declareAction("createModel", CreateModelParams{}, createModelResult{})

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

type createModelResult struct {
	Sortf     int           `json:"sortf"`
	Did       int           `json:"did"`
	LatexPre  string        `json:"latexPre"`
	LatexPost string        `json:"latexPost"`
	Mod       int           `json:"mod"`
	Usn       int           `json:"usn"`
	Vers      []interface{} `json:"vers"`
	Type      int           `json:"type"`
	Css       string        `json:"css"`
	Name      string        `json:"name"`
	Flds      []struct {
		Name   string        `json:"name"`
		Ord    int           `json:"ord"`
		Sticky bool          `json:"sticky"`
		Rtl    bool          `json:"rtl"`
		Font   string        `json:"font"`
		Size   int           `json:"size"`
		Media  []interface{} `json:"media"`
	} `json:"flds"`
	Tmpls []struct {
		Name  string      `json:"name"`
		Ord   int         `json:"ord"`
		Qfmt  string      `json:"qfmt"`
		Afmt  string      `json:"afmt"`
		Did   interface{} `json:"did"`
		Bqfmt string      `json:"bqfmt"`
		Bafmt string      `json:"bafmt"`
	} `json:"tmpls"`
	Tags []interface{}   `json:"tags"`
	Id   string          `json:"id"`
	Req  [][]interface{} `json:"req"`
}
