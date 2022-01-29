package main

import (
	"anki-rest-enhancer/enhancerconf"
	"anki-rest-enhancer/tts"
	_ "embed"
	"encoding/base64"
	"github.com/joomcode/errorx"
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

//go:embed anki-enhancer.yaml
var embeddedConf []byte

func main() {
	err := doMain()
	if err != nil {
		log.Printf("Failed with error %+v", err)
		log.Printf("Exit with status 1")
		os.Exit(1)
	}
	log.Printf("Completed.")
}

func doMain() error {
	var rawConf enhancerconf.YAML
	if err := yaml.UnmarshalStrict(embeddedConf, &rawConf); err != nil {
		return errorx.IllegalFormat.Wrap(err, "Malformed enhancer config")
	}
	conf, err := rawConf.Parse()
	if err != nil {
		return err
	}

	results := tts.TextToSpeech(conf.Azure, map[string]struct{}{"¡Hola, amigo! ¿Qué onda?": {}})
	if err := results.Error; err != nil {
		return err
	}
	for text, result := range results.TextToSpeech {
		if err := result.Error; err != nil {
			log.Printf("Audio generation for text %q failed with error %+v", text, err)
			continue
		}
		log.Printf("Audio for text %q: %q", text, base64.StdEncoding.EncodeToString(result.AudioData))
	}
	return nil
}
