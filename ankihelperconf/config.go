package ankihelperconf

import (
	"net/url"
	"regexp"
	"strings"
	"text/template"
	"time"
)

type Config struct {
	// Path is the path to the file from which this config was loaded.
	Path string

	// RunConfigs contains a list of configurations to be executed.
	// If it's set, fields below should not be used.
	RunConfigs []Config

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
	NoteProcessing    []NoteProcessingRule
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
	Name      *template.Template
	ForFields []AnkiNoteField
	Front     *template.Template
	Back      *template.Template
}

type NotesOrganizationRule struct {
	NotesFilter    string
	TargetDeckName string
}

type NoteProcessingRule struct {
	NoteFilter                string
	MinPauseBetweenExecutions time.Duration
	Timeout                   time.Duration

	Exec NoteProcessingExec
}

type NoteProcessingExec struct {
	Command string
	Args    []NoteProcessingExecArg
	Stdin   NoteProcessingExecArg
}

type NoteProcessingExecArg struct {
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

type replaceProcessor struct {
	pattern     string
	replacement string
}

func (p replaceProcessor) Process(text string) string {
	return strings.ReplaceAll(text, p.pattern, p.replacement)
}
