package ankiconnect

import (
	"anki-rest-enhancer/ankihelperconf"
	"anki-rest-enhancer/util/base64x"
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
	"strings"
	"time"
)

func NewAPI(conf ankihelperconf.Anki) *api {
	client := &http.Client{Timeout: conf.RequestTimeout}
	if conf.LogRequests {
		client.Transport = httputil.NewLoggingRoundTripper(http.DefaultTransport)
	}

	return &api{
		url:    conf.ConnectURL,
		client: client,
	}
}

type api struct {
	url    *url.URL
	client *http.Client
}

var _ API = (*api)(nil)

func (api api) FindNotes(query string) ([]NoteID, error) {
	result, err := api.doReq(findNotesParams{Query: query}, 5)
	if err != nil {
		return nil, err
	}
	return result.(findNotesResult), nil
}

func (api api) FindCards(query string) ([]CardID, error) {
	result, err := api.doReq(findCardsParams{Query: query}, 5)
	if err != nil {
		return nil, err
	}
	return result.(findCardsResult), nil
}

type NoteInfo struct {
	ID     NoteID
	Fields map[string]string
	Tags   []string
}

func (api api) NotesInfo(noteIDs []NoteID) (map[NoteID]NoteInfo, error) {
	if len(noteIDs) == 0 {
		return nil, nil
	}

	rawResult, err := api.doReq(notesInfoParams{NoteIDs: noteIDs}, 5)
	if err != nil {
		return nil, err
	}
	result := rawResult.(notesInfoResult)

	notes := make(map[NoteID]NoteInfo, len(result))
	for _, noteInfo := range result {
		note := NoteInfo{
			ID:     noteInfo.NoteID,
			Fields: map[string]string{},
		}
		for name, value := range noteInfo.Fields {
			note.Fields[name] = value.Value
		}
		note.Tags = noteInfo.Tags
		notes[noteInfo.NoteID] = note
	}
	return notes, nil
}

func (api api) UpdateNoteFields(noteID NoteID, fields map[string]FieldUpdate) error {
	if len(fields) == 0 {
		return nil
	}

	params := updateNoteFieldsParams{Note: updateNoteFieldsNote{
		ID:     noteID,
		Fields: make(map[string]string, len(fields)),
	}}
	for field, fieldUpdate := range fields {
		switch {
		case fieldUpdate.Value != nil:
			params.Note.Fields[field] = *fieldUpdate.Value
		case len(fieldUpdate.AudioData) > 0:
			// AnkiConnect first sets plain values of the fields and only then processes media,
			// which it just adds to the field.
			// Thus, we ask it both to set the field to empty string and then to add audio to the field,
			// achieving 'set field to audio' behaviour instead of simply 'add audio to the field'.
			params.Note.Fields[field] = ""

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

	_, err := api.doReq(params, 5)
	return err
}

func (api api) ModelNames() ([]string, error) {
	rawResult, err := api.doReq(modelNamesParams{}, 5)
	if err != nil {
		return nil, err
	}
	return rawResult.(modelNamesResult), nil
}

func (api api) CreateModel(params CreateModelParams) error {
	_, err := api.doReq(params, 1) // NOTE: this request is not idempotent so it should not be retried
	if err != nil {
		return err
	}
	return nil
}

func (api api) ChangeDeck(deckName string, cardIDs []CardID) error {
	_, err := api.doReq(changeDeckParams{Deck: deckName, Cards: cardIDs}, 5)
	if err != nil {
		return err
	}
	return nil
}

func (api api) AddTags(noteIDs []NoteID, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	params := addTagsParams{
		Notes: noteIDs,
		Tags:  strings.Join(tags, " "),
	}
	_, err := api.doReq(params, 5)
	return err
}

func (api api) doReq(params interface{}, maxAttempts int) (interface{}, error) {
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

	resp, err := api.doReqWithBodyAndRetry(marshalled, maxAttempts)
	if err != nil {
		return nil, err
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

func (api api) doReqWithBodyAndRetry(body []byte, maxAttempts int) (*http.Response, error) {
	if maxAttempts <= 0 {
		panic(errorx.IllegalArgument.New("expected positive maxAttempts but got %d", maxAttempts))
	}

	var errs []error
	retryDelay := 100 * time.Millisecond
	for i := 0; i < maxAttempts; i++ {
		if i > 0 {
			log.Printf("Retrying Anki request error in %s...", retryDelay)
			time.Sleep(retryDelay)
			retryDelay = time.Duration(1.5*float64(retryDelay.Milliseconds())) * time.Millisecond
		}
		resp, err := api.doReqWithBody(body)
		if err == nil {
			return resp, nil
		}
		errs = append(errs, err)
	}
	return nil, errorx.WrapMany(errorx.ExternalError, fmt.Sprintf("Failed to do Anki request %d times", maxAttempts), errs...)
}

func (api api) doReqWithBody(reqBody []byte) (*http.Response, error) {
	req := &http.Request{
		Method: http.MethodPost,
		URL:    api.url,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		ContentLength: int64(len(reqBody)),
		Body:          io.NopCloser(bytes.NewReader(reqBody)),
		GetBody: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(reqBody)), nil
		},
	}

	resp, err := api.client.Do(req)
	if err != nil {
		if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
			return nil, errorx.TimeoutElapsed.Wrap(err, "Text-to-speech api request timed out")
		}
		return nil, errorx.ExternalError.Wrap(err, "Anki API request failed")
	}

	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errorx.ExternalError.Wrap(err, "Failed to read response body")
	}

	resp.Body = io.NopCloser(bytes.NewReader(respBody))
	return resp, nil
}

func (api api) StoreMediaFile(fileName string, fileData io.Reader, deleteExisting bool) error {
	dataBase64, err := base64x.ReadAllEncodeToString(base64.StdEncoding, fileData)
	if err != nil {
		return errorx.Decorate(err, "failed to encode file content to base64")
	}

	params := storeMediaFileParams{
		FileName:       fileName,
		DeleteExisting: deleteExisting,
		DataBase64:     dataBase64,
	}
	_, err = api.doReq(params, 5)
	return err
}
