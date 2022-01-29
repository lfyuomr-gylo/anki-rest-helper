package enhancerconf

import (
	"github.com/joomcode/errorx"
	"log"
	"net/url"
	"time"
)

type YAML struct {
	AnkiConnectAddress string `yaml:"ankiConnectAddress"`

	AzureAPIKey           string  `yaml:"azureApiKey"`
	AzureRegion           string  `yaml:"azureRegion"`
	AzureVoice            *string `yaml:"azureVoice"`
	AzureSynthesisTimeout string  `yaml:"ttsTimeout"`
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

	if key := c.AzureAPIKey; key == "" {
		return Config{}, errorx.IllegalState.New("Azure API key must be specified")
	} else {
		conf.AzureAPIKey = key
	}

	if region := c.AzureRegion; region == "" {
		return Config{}, errorx.IllegalState.New("Azure region must be specified")
	} else {
		conf.AzureRegion = region
	}

	conf.AzureVoice = c.AzureVoice

	{
		const defaultAzureSynthesisTimeout = "30s"
		timeout := c.AzureSynthesisTimeout
		if timeout == "" {
			log.Printf("Azure Synthesis Timeout is not specified in the config. Use default: %s", defaultAzureSynthesisTimeout)
			timeout = defaultAzureSynthesisTimeout
		}
		parsed, err := time.ParseDuration(timeout)
		if err != nil {
			return Config{}, errorx.IllegalFormat.Wrap(err, "Malformed speech synthesis timeout: %s", timeout)
		}
		if parsed <= 0 {
			return Config{}, errorx.IllegalState.New("Speech synthesis timeout must be positive")
		}
		conf.AzureSynthesisTimeout = parsed
	}
}

type Config struct {
	AnkiConnectAddress *url.URL

	AzureAPIKey           string
	AzureRegion           string
	AzureVoice            *string
	AzureSynthesisTimeout time.Duration
}
