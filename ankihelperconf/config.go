package ankihelperconf

import (
	"net/url"
	"regexp"
	"time"
)

type Config struct {
	Anki    Anki
	Azure   Azure
	Actions Actions
}

type Azure struct {
	APIKey                  string
	EndpointURL             *url.URL
	Voice                   string
	RequestTimeout          time.Duration
	Language                string
	MinPauseBetweenRequests time.Duration

	LogRequests            bool
	RetryOnTooManyRequests bool
	MaxRetries             int
}

type Anki struct {
	ConnectURL     *url.URL
	RequestTimeout time.Duration
	LogRequests    bool
}

type Actions struct {
	TTS       []AnkiTTS
	NoteTypes []AnkiNoteType
}

type AnkiTTS struct {
	// oneof:
	Fields                *AnkiTTSFields
	GeneratedNoteTypeName *string

	TextPreprocessors []TextProcessor
}

type AnkiTTSFields struct {
	NoteFilter, TextField, AudioField string
}

type AnkiNoteType struct {
	Name      string
	CSS       string
	Fields    []AnkiNoteField
	Templates []AnkiCardTemplate
}

type AnkiNoteField YAMLAnkiNoteField

type AnkiCardTemplate struct {
	Name      string
	ForFields []AnkiNoteField
	Front     string
	Back      string
}

type TextProcessor interface {
	Process(text string) string
}

type regexpProcessor struct {
	regexp      *regexp.Regexp
	replacement string
}

func (p regexpProcessor) Process(text string) string {
	return p.regexp.ReplaceAllString(text, p.replacement)
}
