package main

import (
	"anki-rest-enhancer/ankiconnect"
	"anki-rest-enhancer/ankihelper"
	"anki-rest-enhancer/ankihelperconf"
	"anki-rest-enhancer/azuretts"
	"anki-rest-enhancer/noteprocessing"
	"encoding/json"
	"flag"
	"github.com/joomcode/errorx"
	"log"
	"os"
	"path/filepath"
)

var flagConfigPath = flag.String("config", "", "path to config file")
var flagPrintConfig = flag.Bool("print-config", false, "whether the internal representation of the config should be printed once it's loaded")
var flagNoOp = flag.Bool("noop", false, "if this flag is set to true, tool exits after the config is loaded (and optionally printed)")

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

	conf, err := ankihelperconf.LoadYAML(configPath)
	if err != nil {
		return err
	}

	if *flagPrintConfig {
		if err := printConfig(conf); err != nil {
			return err
		}
	}
	if *flagNoOp {
		return nil
	}

	return runConfig(conf)
}

func printConfig(conf ankihelperconf.Config) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(conf); err != nil {
		return errorx.Decorate(err, "failed to dump internal config representation")
	}
	return nil
}

func runConfig(conf ankihelperconf.Config) error {
	log.Printf("Running config file %s", conf.Path)
	if len(conf.RunConfigs) > 0 {
		for _, conf := range conf.RunConfigs {
			if err := runConfig(conf); err != nil {
				return err
			}
		}
		return nil
	}

	azureTTS := azuretts.NewAPI(conf.Azure)
	ankiConnect := ankiconnect.NewAPI(conf.Anki)
	scriptRunner := noteprocessing.NewScriptRunner()
	enhancer := ankihelper.NewHelper(ankiConnect, azureTTS, scriptRunner)
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
