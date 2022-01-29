package main

import (
	"anki-rest-enhancer/enhance"
	"anki-rest-enhancer/enhancerconf"
	_ "embed"
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

	enhancer := enhance.NewEnhancer(conf)
	return enhancer.Enhance(conf)
}
