package tts

import (
	"github.com/stretchr/testify/require"
	"net/url"
	"os"
	"testing"
)

func TestFo(t *testing.T) {
	endpointURL, err := url.Parse("https://switzerlandnorth.tts.speech.microsoft.com/cognitiveservices/v1")
	require.NoError(t, err)
	req, err := makeTextToSpeechRequest(apiCallArgs{
		APIKey:      "foo",
		Language:    "es-PE",
		Voice:       "es-PE-AlexNeural",
		Text:        "¡Hola, amigo! ¿Qué ondo?",
		EndpointURL: endpointURL,
	})
	require.NoError(t, err)
	require.NoError(t, req.Write(os.Stdout))
}
