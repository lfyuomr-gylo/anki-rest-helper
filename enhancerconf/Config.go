package enhancerconf

import (
	"github.com/joomcode/errorx"
	"log"
	"net/url"
	"strings"
	"time"
)

type YAML struct {
	AnkiConnectAddress string    `yaml:"ankiConnectAddress"`
	Azure              YAMLAzure `yaml:"azure"`
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
		conf.TTSEndpoint = parsed
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

func (c YAML) Parse() (Config, error) {
	conf := Config{}

	{
		const defaultAnkiConnectAddress = "http://localhost:8765"
		addr := c.AnkiConnectAddress
		if addr == "" {
			log.Printf("AnkiConnect address is not specified in the config. Use default: %s", defaultAnkiConnectAddress)
			addr = defaultAnkiConnectAddress
		}
		parsed, err := url.Parse(addr)
		if err != nil {
			return Config{}, errorx.IllegalFormat.Wrap(err, "Malformed AnkiConnect address")
		}
		conf.AnkiConnectAddress = parsed
	}

	{
		azureConf, err := c.Azure.Parse()
		if err != nil {
			return Config{}, errorx.Decorate(err, "invalid Azure config")
		}
		conf.Azure = azureConf
	}

	return conf, nil
}

type Config struct {
	AnkiConnectAddress *url.URL
	Azure              Azure
}

type Azure struct {
	APIKey         string
	TTSEndpoint    *url.URL
	Voice          string
	RequestTimeout time.Duration
	Language       string

	LogRequests bool
}
