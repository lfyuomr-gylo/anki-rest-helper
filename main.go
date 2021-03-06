package main

import (
	"anki-rest-enhancer/ankiconnect"
	"anki-rest-enhancer/ankihelper"
	"anki-rest-enhancer/ankihelperconf"
	"anki-rest-enhancer/azuretts"
	"flag"
	"github.com/joomcode/errorx"
	"gopkg.in/yaml.v2"
	"io"
	"log"
	"os"
	"path/filepath"
)

var flagConfigPath = flag.String("config", "", "path to config file")

func main() {
	flag.Parse()

	err := doMain()
	if err != nil {
		log.Printf("Failed with error %+v", err)
		log.Printf("Exit with status 1")
		os.Exit(1)
	}
	log.Printf("Completed.")
}

func doMain() error {
	configPath := findConfigFile()
	confFile, err := os.Open(configPath)
	if err != nil {
		return errorx.ExternalError.Wrap(err, "failed to open config file: %s", configPath)
	}
	defer func() { _ = confFile.Close() }()
	confData, err := io.ReadAll(confFile)
	if err != nil {
		return errorx.ExternalError.Wrap(err, "failed to read config file")
	}

	var rawConf ankihelperconf.YAML
	if err := yaml.UnmarshalStrict(confData, &rawConf); err != nil {
		return errorx.IllegalFormat.Wrap(err, "Malformed enhancer config")
	}
	conf, err := rawConf.Parse(filepath.Dir(configPath))
	if err != nil {
		return err
	}

	azureTTS := azuretts.NewAPI(conf.Azure)
	ankiConnect := ankiconnect.NewAPI(conf.Anki)

	enhancer := ankihelper.NewHelper(ankiConnect, azureTTS)
	return enhancer.Run(conf.Actions)
}

func findConfigFile() string {
	if path := *flagConfigPath; path != "" {
		log.Printf("Use config path from CLI arguments: %s", path)
		return path
	}

	var dirs []string
	for _, source := range []struct {
		dirType string
		getDir  func() (string, error)
	}{
		{"current directory", os.Getwd},
		{"user config directory", os.UserConfigDir},
		{"user home directory", os.UserHomeDir},
	} {
		dir, err := source.getDir()
		if err != nil {
			log.Printf("Failed to get %s: %+v", source.dirType, err)
		}
		dirs = append(dirs, dir)
	}

	const defaultConfigFileName = "anki-helper.yaml"
	for _, dir := range dirs {
		path := filepath.Join(dir, defaultConfigFileName)
		log.Printf("Check for config file at %s", path)
		if info, err := os.Lstat(path); err == nil && info.Mode().IsRegular() {
			log.Printf("Use configuration from %s", path)
			return path
		}
	}
	panic(errorx.Panic(errorx.IllegalState.New("Failed to find")))
}
