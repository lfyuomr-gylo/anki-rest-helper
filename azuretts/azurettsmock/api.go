package azurettsmock

import (
	"anki-rest-enhancer/azuretts"
	"github.com/joomcode/errorx"
)

type API struct {
	TextToSpeechFunc func(texts map[string]struct{}) map[string]azuretts.TextToSpeechResult
}

var _ azuretts.API = (*API)(nil)

func (api *API) Reset() {
	*api = API{}
}

func (api *API) TextToSpeech(texts map[string]struct{}) map[string]azuretts.TextToSpeechResult {
	if behaviour := api.TextToSpeechFunc; behaviour != nil {
		return behaviour(texts)
	}
	panic(errorx.Panic(errorx.NotImplemented.New("Mock behaviour is not specified for method TextToSpeech")))
}
