package enhancerconf

import (
	"fmt"
	"github.com/joomcode/errorx"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type YAML struct {
	Anki  YAMLAnki  `yaml:"anki"`
	Azure YAMLAzure `yaml:"azure"`
}

func (c YAML) Parse(configDir string) (Config, error) {
	conf := Config{}

	{
		azureConf, err := c.Azure.Parse(configDir)
		if err != nil {
			return Config{}, errorx.Decorate(err, "invalid Azure config")
		}
		conf.Azure = azureConf
	}

	{
		ankiConf, err := c.Anki.Parse()
		if err != nil {
			return Config{}, errorx.Decorate(err, "invalid Anki config")
		}
		conf.Anki = ankiConf
	}

	return conf, nil
}

type YAMLAzure struct {
	// required:
	APIKey      string `yaml:"apiKey"`
	APIKeyFile  string `yaml:"apiKeyFile"`
	EndpointURL string `yaml:"endpointUrl"`
	Voice       string `yaml:"voice"`

	// optional:
	LogRequests             bool   `yaml:"logRequests"`
	Language                string `yaml:"language"`
	RequestTimeout          string `yaml:"requestTimeout"`
	MinPauseBetweenRequests string `yaml:"minPauseBetweenRequests"`
}

func (c YAMLAzure) Parse(configDir string) (Azure, error) {
	var conf Azure
	if key := c.APIKey; key != "" {
		conf.APIKey = key
	} else if keyPath := c.APIKeyFile; keyPath != "" {
		if !filepath.IsAbs(keyPath) {
			keyPath = filepath.Join(configDir, keyPath)
		}

		log.Printf("Loading Azure API Key from %s", keyPath)
		file, err := os.Open(keyPath)
		if err != nil {
			return Azure{}, errorx.ExternalError.Wrap(err, "failed to open file ")
		}
		defer func() { _ = file.Close() }()
		rawKey, err := io.ReadAll(file)
		if err != nil {
			return Azure{}, errorx.ExternalError.Wrap(err, "failed to read Azure API key")
		}
		conf.APIKey = strings.TrimSpace(string(rawKey))
	} else {
		return Azure{}, errorx.IllegalState.New("API Key is not specified")

	}

	if endpoint := c.EndpointURL; endpoint == "" {
		return Azure{}, errorx.IllegalState.New("Endpoint URL is not specified")
	} else {
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return Azure{}, errorx.IllegalFormat.Wrap(err, "Malformed Azure endpoint: %q", endpoint)
		}
		conf.EndpointURL = parsed
	}

	if voice := c.Voice; voice == "" {
		return Azure{}, errorx.IllegalState.New("Voice is not specified")
	} else {
		conf.Voice = voice
	}

	if lang := c.Language; lang == "" {
		log.Println("Text-to-speech language is not explicitly specified in the config. Trying to infer from voice name...")
		langLocaleVoice := strings.SplitN(c.Voice, "-", 3)
		if len(langLocaleVoice) != 3 {
			return Azure{}, errorx.IllegalFormat.New("Faile to infer language from voice name. Expected <lang-locale-voice> but got %q", c.Voice)
		}
		conf.Language = langLocaleVoice[0] + "-" + langLocaleVoice[1]
	} else {
		conf.Language = c.Language
	}

	{
		const defaultRequestTimeout = "30s"
		timeout := c.RequestTimeout
		if timeout == "" {
			log.Println("Azure request timeout is not specified, use default %q", defaultRequestTimeout)
			timeout = defaultRequestTimeout
		}
		parsed, err := time.ParseDuration(timeout)
		if err != nil {
			return Azure{}, errorx.IllegalFormat.Wrap(err, "malformed request timeout")
		}
		conf.RequestTimeout = parsed
	}

	conf.LogRequests = c.LogRequests

	{
		const defaultMinPauseBetweenRequests = "1s"
		pause := c.MinPauseBetweenRequests
		if c.MinPauseBetweenRequests == "" {
			log.Printf("Minimum pause between requests to Azure API is not set. Use default %q", pause)
			c.MinPauseBetweenRequests = defaultMinPauseBetweenRequests
		}
		parsed, err := time.ParseDuration(pause)
		if err != nil {
			return Azure{}, errorx.IllegalFormat.New("Failed to parse minimum pause between requests to Azure API: %q", parsed)
		}
		conf.MinPauseBetweenRequests = parsed
	}

	return conf, nil
}

type YAMLAnki struct {
	ConnectURL     string             `yaml:"connectUrl"`
	RequestTimeout string             `yaml:"requestTimeout"`
	TTS            []YAMLAnkiTTS      `yaml:"tts"`
	NoteTypes      []YAMLAnkiNoteType `yaml:"noteTypes"`
	LogRequests    bool               `yaml:"logRequests"`
}

func (c YAMLAnki) Parse() (Anki, error) {
	var conf Anki

	{
		const defaultAnkiConnectAddress = "http://localhost:8765"
		addr := c.ConnectURL
		if addr == "" {
			log.Printf("AnkiConnect address is not specified in the config. Use default: %s", defaultAnkiConnectAddress)
			addr = defaultAnkiConnectAddress
		}
		parsed, err := url.Parse(addr)
		if err != nil {
			return Anki{}, errorx.IllegalFormat.Wrap(err, "Malformed AnkiConnect address")
		}
		conf.ConnectURL = parsed
	}

	{
		const defaultAnkiRequestTimeout = "30s"
		timeout := c.RequestTimeout
		if timeout == "" {
			log.Printf("Anki request timeout is not specified in the config. Use default timeout %q", defaultAnkiRequestTimeout)
			timeout = defaultAnkiRequestTimeout
		}
		parsed, err := time.ParseDuration(timeout)
		if err != nil {
			return Anki{}, errorx.IllegalFormat.Wrap(err, "malformed Anki request timeout")
		}
		conf.RequestTimeout = parsed
	}

	for i, tts := range c.TTS {
		parsed, err := tts.Parse()
		if err != nil {
			return Anki{}, errorx.Decorate(err, "invalid tts #%d", i)
		}
		conf.TTS = append(conf.TTS, parsed)
	}

	for i, noteType := range c.NoteTypes {
		parsed, err := noteType.Parse()
		if err != nil {
			return Anki{}, errorx.Decorate(err, "invalid note type #%d", i)
		}
		conf.NoteTypes = append(conf.NoteTypes, parsed)
		// TODO: make sure note type names are unique
	}

	conf.LogRequests = c.LogRequests

	return conf, nil
}

type YAMLAnkiTTS struct {
	// required:
	TextField  string `yaml:"textField"`
	AudioField string `yaml:"audioField"`

	// optional:
	NoteFilter     string               `yaml:"noteFilter"`
	TextProcessing []YAMLTextProcessing `yaml:"textPreprocessing"`
}

func (c YAMLAnkiTTS) Parse() (AnkiTTS, error) {
	var conf AnkiTTS

	if tf := c.TextField; tf == "" {
		return AnkiTTS{}, errorx.IllegalState.New("Text field must be specified for TTS")
	} else {
		conf.TextField = tf
	}

	if af := c.AudioField; af == "" {
		return AnkiTTS{}, errorx.IllegalState.New("Audio field must be specified for TTS")
	} else {
		conf.AudioField = af
	}

	if filter := c.NoteFilter; filter == "" {
		defaultFilter := fmt.Sprintf(`"%s:_*" "%s:"`, c.TextField, c.AudioField)
		log.Printf("No filter specified in TTS for fields %s->%s. Infer filter: %s", c.TextField, c.AudioField, defaultFilter)
		conf.NoteFilter = defaultFilter
	} else {
		conf.NoteFilter = filter
	}

	for i, processing := range c.TextProcessing {
		parsed, err := processing.Parse()
		if err != nil {
			return AnkiTTS{}, errorx.Decorate(err, "Invalid TTS #%d", i)
		}
		conf.TextPreprocessors = append(conf.TextPreprocessors, parsed)
	}

	return conf, nil
}

type YAMLTextProcessing struct {
	Regexp      string `yaml:"regexp"`
	Replacement string `yaml:"replacement"`
}

func (c YAMLTextProcessing) Parse() (TextProcessor, error) {
	compiled, err := regexp.Compile(c.Regexp)
	if err != nil {
		return nil, errorx.IllegalFormat.Wrap(err, "malformed regexp")
	}
	return regexpProcessor{regexp: compiled, replacement: c.Replacement}, nil
}

type YAMLAnkiNoteType struct {
	Name      string                 `yaml:"name"`
	CSS       string                 `yaml:"css"`
	Fields    []YAMLAnkiNoteField    `yaml:"fields"`
	Templates []YAMLAnkiCardTemplate `yaml:"templates"`
}

func (t YAMLAnkiNoteType) Parse() (AnkiNoteType, error) {
	if err := ValidateName(t.Name); err != nil {
		return AnkiNoteType{}, err
	}

	// TODO: ensure fields have unique names
	fields := make([]AnkiNoteField, len(t.Fields))
	fieldsByName := make(map[string]AnkiNoteField, len(t.Fields))
	for i, field := range t.Fields {
		parsed, err := field.Parse()
		if err != nil {
			return AnkiNoteType{}, errorx.Decorate(err, "invalid field #%d", i)
		}

		if _, ok := fieldsByName[parsed.Name]; ok {
			return AnkiNoteType{}, errorx.IllegalState.New("field %q is duplicated", parsed.Name)
		}

		fields[i] = parsed
		fieldsByName[parsed.Name] = parsed
	}

	// TODO: ensure templates have unique names
	templates := make([]AnkiCardTemplate, len(t.Templates))
	for i, template := range t.Templates {
		parsed, err := template.Parse(fieldsByName)
		if err != nil {
			return AnkiNoteType{}, errorx.Decorate(err, "invalid card template #%d", i)
		}
		templates[i] = parsed
	}

	return AnkiNoteType{
		Name:      t.Name,
		CSS:       t.CSS,
		Fields:    fields,
		Templates: templates,
	}, nil
}

type YAMLAnkiNoteField struct {
	Name          string `yaml:"name"`
	SkipExample   bool   `yaml:"skipExample"`
	SkipVoiceover bool   `yaml:"skipVoiceover"`
}

func (f YAMLAnkiNoteField) Parse() (AnkiNoteField, error) {
	if err := ValidateName(f.Name); err != nil {
		return AnkiNoteField{}, err
	}
	return AnkiNoteField(f), nil
}

type YAMLAnkiCardTemplate struct {
	Name      string   `yaml:"name"`
	ForFields []string `yaml:"forFields"`
	Front     string   `yaml:"front"`
	Back      string   `yaml:"back"`
}

func (t YAMLAnkiCardTemplate) Parse(fieldsByName map[string]AnkiNoteField) (AnkiCardTemplate, error) {
	fields := make([]AnkiNoteField, 0, len(t.ForFields))
	for _, fieldName := range t.ForFields {
		field, ok := fieldsByName[fieldName]
		if !ok {
			return AnkiCardTemplate{}, errorx.IllegalState.New("there is no field %q", fieldName)
		}
		fields = append(fields, field)
	}

	return AnkiCardTemplate{
		Name:      t.Name,
		ForFields: fields,
		Front:     t.Front,
		Back:      t.Back,
	}, nil
}

var namePattern = regexp.MustCompile(`^[A-Za-z_]\w*$`)

func ValidateName(name string) error {
	if ok := namePattern.MatchString(name); !ok {
		return errorx.IllegalFormat.New("malformed name. Expected a valid variable name but got: %q", name)
	}
	return nil
}

type Config struct {
	Anki  Anki
	Azure Azure
}

type Azure struct {
	APIKey                  string
	EndpointURL             *url.URL
	Voice                   string
	RequestTimeout          time.Duration
	Language                string
	MinPauseBetweenRequests time.Duration

	LogRequests bool
}

type Anki struct {
	ConnectURL     *url.URL
	RequestTimeout time.Duration
	TTS            []AnkiTTS
	NoteTypes      []AnkiNoteType
	LogRequests    bool
}

type AnkiTTS struct {
	NoteFilter, TextField, AudioField string
	TextPreprocessors                 []TextProcessor
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
