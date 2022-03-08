package ankihelperconf

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
	Anki    YAMLAnki    `yaml:"anki"`
	Azure   YAMLAzure   `yaml:"azure"`
	Actions YAMLActions `yaml:"actions"`
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

	{
		Actions, err := c.Actions.Parse()
		if err != nil {
			return Config{}, errorx.Decorate(err, "invalid Actions config")
		}
		conf.Actions = Actions
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
	RetryOnTooManyRequests  bool   `yaml:"retryOnTooManyRequests"`
	MaxRetries              *int   `yaml:"maxRetries"`
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
			log.Printf("Azure request timeout is not specified, use default %q", defaultRequestTimeout)
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

	conf.RetryOnTooManyRequests = c.RetryOnTooManyRequests
	{
		const defaultMaxRetries = 5
		maxRetries := defaultMaxRetries
		if override := c.MaxRetries; override != nil {
			if *override <= 0 {
				return Azure{}, errorx.IllegalState.New("Max retries number must be positive")
			}
			maxRetries = *override
		}
		conf.MaxRetries = maxRetries
	}

	return conf, nil
}

type YAMLAnki struct {
	ConnectURL     string `yaml:"connectUrl"`
	RequestTimeout string `yaml:"requestTimeout"`
	LogRequests    bool   `yaml:"logRequests"`
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

	conf.LogRequests = c.LogRequests

	return conf, nil
}

type YAMLActions struct {
	TTS       []YAMLAnkiTTS      `yaml:"tts"`
	NoteTypes []YAMLAnkiNoteType `yaml:"noteTypes"`
}

func (e YAMLActions) Parse() (Actions, error) {
	var actions Actions

	for i, tts := range e.TTS {
		parsed, err := tts.Parse()
		if err != nil {
			return Actions{}, errorx.Decorate(err, "invalid tts #%d", i)
		}
		actions.TTS = append(actions.TTS, parsed)
	}

	for i, noteType := range e.NoteTypes {
		parsed, err := noteType.Parse()
		if err != nil {
			return Actions{}, errorx.Decorate(err, "invalid note type #%d", i)
		}
		actions.NoteTypes = append(actions.NoteTypes, parsed)
	}

	return actions, nil
}

type YAMLAnkiTTS struct {
	ForGeneratedNoteType string `yaml:"forGeneratedNoteType"`
	TextField            string `yaml:"textField"`
	AudioField           string `yaml:"audioField"`

	// optional:
	NoteFilter     string               `yaml:"noteFilter"`
	TextProcessing []YAMLTextProcessing `yaml:"textPreprocessing"`
}

func (c YAMLAnkiTTS) Parse() (AnkiTTS, error) {
	var conf AnkiTTS

	switch {
	case c.ForGeneratedNoteType != "" && c.TextField == "" && c.AudioField == "":
		if c.NoteFilter != "" {
			return AnkiTTS{}, errorx.IllegalState.New("Note Filter doesn't make sense for generated note types")
		}

		noteType := c.ForGeneratedNoteType
		conf.GeneratedNoteTypeName = &noteType
	case c.ForGeneratedNoteType == "" && c.TextField != "" && c.AudioField != "":
		noteFilter := c.NoteFilter
		if noteFilter == "" {
			defaultFilter := fmt.Sprintf(`"%s:_*" "%s:"`, c.TextField, c.AudioField)
			log.Printf("No filter specified in TTS for fields %s->%s. Infer filter: %s", c.TextField, c.AudioField, defaultFilter)
			noteFilter = defaultFilter
		}

		conf.Fields = &AnkiTTSFields{
			NoteFilter: noteFilter,
			TextField:  c.TextField,
			AudioField: c.AudioField,
		}
	default:
		return AnkiTTS{}, errorx.IllegalState.New("Either generated note type or both text and audio fields must be specified for TTS")
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
	Name          string            `yaml:"name"`
	SkipExample   bool              `yaml:"skipExample"`
	SkipVoiceover bool              `yaml:"skipVoiceover"`
	Vars          map[string]string `yaml:"vars"`
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
