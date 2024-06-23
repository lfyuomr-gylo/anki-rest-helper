package main

import (
	"anki-rest-enhancer/noteprocessing"
	"anki-rest-enhancer/util/iox"
	"anki-rest-enhancer/util/lang"
	"context"
	"encoding/json"
	"flag"
	"github.com/google/generative-ai-go/genai"
	"github.com/joomcode/errorx"
	"google.golang.org/api/option"
	"log"
	"os"
	"strings"
)

type Params struct {
	APIKeyFile string
	ModelID    string
	Field      string
	Prompt     string
	Tag        string
}

func mustParseArgs() Params {
	var params Params
	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.StringVar(&params.APIKeyFile, "api-key-file", "", "path to the file that contains the Gemini API key")
	fs.StringVar(&params.ModelID, "model", "gemini-1.0-pro", "the ID of the Gemini model to use")
	fs.StringVar(&params.Field, "field", "", "name of the target note field that should be set")
	fs.StringVar(&params.Prompt, "prompt", "", "text that should be sent to Gemini to generate the field")
	fs.StringVar(&params.Tag, "add-tag", "", "optional tag ")
	_ = fs.Parse(os.Args[1:]) // ignore error as it's never returned due to ExitOnError setting

	if params.APIKeyFile == "" {
		log.Fatalf("API Key must be provided")
	}
	if params.Field == "" {
		log.Fatalf("Target note field must be provided")
	}
	if params.Prompt == "" {
		log.Fatalf("Prompt must be provided")
	}
	return params
}

func main() {
	ctx := context.Background()
	params := mustParseArgs()

	modifications, err := doMain(ctx, params)
	if err != nil {
		log.Fatalf("Execution failed: %+v", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(modifications); err != nil {
		log.Fatalf("Failed to write modifications to STDOUT: %+v", err)
	}
}

func doMain(ctx context.Context, params Params) ([]noteprocessing.Modification, error) {
	apiKey, err := os.ReadFile(params.APIKeyFile)
	if err != nil {
		return nil, errorx.ExternalError.New("failed to read API key file %q: %+v", params.APIKeyFile, err)
	}
	client, err := genai.NewClient(ctx, option.WithAPIKey(string(apiKey)))
	if err != nil {
		return nil, errorx.ExternalError.Wrap(err, "failed to construct a Gemini client")
	}
	defer iox.Close(client)
	model := client.GenerativeModel(params.ModelID)

	content, err := model.GenerateContent(ctx, genai.Text(params.Prompt))
	if err != nil {
		return nil, errorx.ExternalError.Wrap(err, "content generation failed")
	}

	var modifications []noteprocessing.Modification
	generatedContent := string(content.Candidates[0].Content.Parts[0].(genai.Text))
	modifications = append(modifications, noteprocessing.Modification{
		SetField: lang.New(map[string]string{
			params.Field: strings.TrimSpace(generatedContent),
		}),
	})
	if params.Tag != "" {
		modifications = append(modifications, noteprocessing.Modification{
			AddTag: lang.New(params.Tag),
		})
	}
	return modifications, nil
}
