package ankiconnect

import (
	"anki-rest-enhancer/enhancerconf"
	"anki-rest-enhancer/util/httputil"
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/joomcode/errorx"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"reflect"
)

func NewAPI(conf enhancerconf.Anki) *API {
	client := &http.Client{Timeout: conf.RequestTimeout}
	if conf.LogRequests {
		client.Transport = httputil.NewLoggingRoundTripper(http.DefaultTransport)
	}

	return &API{
		url:    conf.ConnectURL,
		client: client,
	}
}

type API struct {
	url    *url.URL
	client *http.Client
}

type NoteID int64

func (api API) FindNotes(query string) ([]NoteID, error) {
	result, err := api.doReq(findNotesParams{Query: query})
	if err != nil {
		return nil, err
	}
	return result.(findNotesResult), nil
}

type NoteInfo struct {
	Fields map[string]string
}

func (api API) NotesInfo(noteIDs []NoteID) (map[NoteID]NoteInfo, error) {
	rawResult, err := api.doReq(notesInfoParams{NoteIDs: noteIDs})
	if err != nil {
		return nil, err
	}
	result := rawResult.(notesInfoResult)

	notes := make(map[NoteID]NoteInfo, len(result))
	for _, noteInfo := range result {
		note := NoteInfo{Fields: map[string]string{}}
		for name, value := range noteInfo.Fields {
			note.Fields[name] = value.Value
		}
		notes[noteInfo.NoteID] = note
	}
	return notes, nil
}

type FieldUpdate struct {
	// one of
	Value     string
	AudioData []byte
}

func (api API) UpdateNoteFields(noteID NoteID, fields map[string]FieldUpdate) error {
	params := updateNoteFieldsParams{Note: updateNoteFieldsNote{
		ID:     noteID,
		Fields: make(map[string]string, len(fields)),
	}}
	for field, fieldUpdate := range fields {
		switch {
		case fieldUpdate.Value != "":
			params.Note.Fields[field] = fieldUpdate.Value
		case len(fieldUpdate.AudioData) > 0:
			fileName := fmt.Sprintf("%x.mp3", md5.Sum(fieldUpdate.AudioData))
			params.Note.Audio = append(params.Note.Audio, updateNoteFieldsAudio{
				FileName:   fileName,
				Base64Data: base64.StdEncoding.EncodeToString(fieldUpdate.AudioData),
				Fields:     []string{field},
			})
		default:
			log.Printf("WARN: %+v", errorx.IllegalState.New("got empty field %q update for note %d", field, noteID))
		}
	}

	_, err := api.doReq(params)
	return err
}

func (api API) ModelNames() ([]string, error) {
	rawResult, err := api.doReq(modelNamesParams{})
	if err != nil {
		return nil, err
	}
	return rawResult.(modelNamesResult), nil
}

func (api API) CreateModel(params CreateModelParams) error {
	_, err := api.doReq(params)
	if err != nil {
		return err
	}
	return nil
}

func (api API) doReq(params interface{}) (interface{}, error) {
	actionName, ok := actionParamsMapping[reflect.TypeOf(params)]
	if !ok {
		panic(errorx.IllegalState.New("got action params of unexpected type: %+v", params))
	}
	marshalled, err := json.Marshal(requestPayload{
		Action:  actionName,
		Version: 6,
		Params:  params,
	})
	if err != nil {
		return nil, errorx.IllegalState.Wrap(err, "failed to marshal AnkiConnect request")
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    api.url,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		ContentLength: int64(len(marshalled)),
		Body:          io.NopCloser(bytes.NewReader(marshalled)),
	}

	resp, err := api.client.Do(req)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return nil, errorx.TimeoutElapsed.Wrap(err, "Text-to-speech API request timed out")
		}
		return nil, errorx.ExternalError.Wrap(err, "Azure API request failed")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errorx.ExternalError.Wrap(err, "failed to read response body")
	}
	if status := resp.StatusCode; status >= 300 {
		return nil, errorx.ExternalError.New("bad response status: %d", status)
	}

	var unmarshalledBody responsePayload
	if err := json.Unmarshal(body, &unmarshalledBody); err != nil {
		return nil, errorx.IllegalFormat.Wrap(err, "failed to unmarshal response body")
	}
	if errStr := unmarshalledBody.Error; errStr != nil {
		return nil, errorx.ExternalError.New("AnkiConnect error: %s", *errStr)
	}

	resultType, ok := actionResultMapping[actionName]
	if !ok {
		panic(errorx.IllegalState.New("failed to find result type for action %q", actionName))
	}
	resultPtrVal := reflect.New(resultType)
	if err := json.Unmarshal(unmarshalledBody.Result, resultPtrVal.Interface()); err != nil {
		return nil, errorx.IllegalFormat.Wrap(err, "Failed to unmarshal action result")
	}
	return resultPtrVal.Elem().Interface(), nil
}
