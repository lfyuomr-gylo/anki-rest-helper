package ankihelperconf

import (
	"anki-rest-enhancer/util/lang/set"
	"net/url"
	"regexp"
	"text/template"
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
	UploadMedia       []AnkiUploadMedia
	TTS               []AnkiTTS
	NoteTypes         []AnkiNoteType
	CardsOrganization []NotesOrganizationRule
	NotesPopulation   []NotesPopulationRule
}

type AnkiUploadMedia struct {
	AnkiName string
	FilePath string
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

type NotesOrganizationRule struct {
	NotesFilter    string
	TargetDeckName string
}

type NotesPopulationRule struct {
	NoteFilter                string
	ProducedFields            set.Set[string]
	MinPauseBetweenExecutions time.Duration

	Exec NotesPopulationExec
}

type NotesPopulationExec struct {
	Command string
	Args    []NotesPopulationExecArg
}

type NotesPopulationExecArg struct {
	// oneof
	PlainString *string
	Template    *template.Template
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
