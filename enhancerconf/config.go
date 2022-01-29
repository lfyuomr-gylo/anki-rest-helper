package enhancerconf

import (
	"fmt"
	"github.com/joomcode/errorx"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type YAML struct {
	Anki  YAMLAnki  `yaml:"anki"`
	Azure YAMLAzure `yaml:"azure"`
}

func (c YAML) Parse() (Config, error) {
	conf := Config{}

	{
		azureConf, err := c.Azure.Parse()
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
	EndpointURL string `yaml:"endpointUrl"`
	Voice       string `yaml:"voice"`

	// optional:
	LogRequests    bool   `yaml:"logRequests"`
	Language       string `yaml:"language"`
	RequestTimeout string `yaml:"requestTimeout"`
}

func (c YAMLAzure) Parse() (Azure, error) {
	var conf Azure

	if key := c.APIKey; key == "" {
		return Azure{}, errorx.IllegalState.New("API Key is not specified")
	} else {
		conf.APIKey = key
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

	return conf, nil
}

type YAMLAnki struct {
	ConnectURL     string        `yaml:"connectUrl"`
	RequestTimeout string        `yaml:"requestTimeout"`
	TTS            []YAMLAnkiTTS `yaml:"tts"`
	LogRequests    bool          `yaml:"logRequests"`
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

type Config struct {
	Anki  Anki
	Azure Azure
}

type Azure struct {
	APIKey         string
	EndpointURL    *url.URL
	Voice          string
	RequestTimeout time.Duration
	Language       string

	LogRequests bool
}

type Anki struct {
	ConnectURL     *url.URL
	RequestTimeout time.Duration
	TTS            []AnkiTTS

	LogRequests bool
}

type AnkiTTS struct {
	NoteFilter, TextField, AudioField string
	TextPreprocessors                 []TextProcessor
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
