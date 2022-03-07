package azuretts

type TextToSpeechResult struct {
	Error    error
	AudioMP3 []byte
}

type API interface {
	// TextToSpeech runs bulk text-to-speech generation for all the specified texts.
	TextToSpeech(texts map[string]struct{}) map[string]TextToSpeechResult
}
