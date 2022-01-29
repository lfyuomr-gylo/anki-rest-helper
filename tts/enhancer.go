package tts

import (
	"anki-rest-enhancer/enhancerconf"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
	"github.com/joomcode/errorx"
	"log"
	"time"
)

type TextToSpeechResults struct {
	// Error is non-nil if the whole speech generation failed (e.g. due to malformed configuration)
	Error        error
	TextToSpeech map[string]TextToSpeechResult
}

type TextToSpeechResult struct {
	Error     error
	AudioData []byte
}

func TextToSpeech(conf enhancerconf.Config, texts map[string]struct{}) TextToSpeechResults {
	speechConf, err := speech.NewSpeechConfigFromSubscription(conf.AzureAPIKey, conf.AzureRegion)
	if err != nil {
		return TextToSpeechResults{Error: errorx.IllegalState.Wrap(err, "failed to construct Speech SDK config")}
	}
	defer speechConf.Close()
	if voice := conf.AzureVoice; voice != nil {
		if err := speechConf.SetSpeechSynthesisVoiceName(*voice); err != nil {
			return TextToSpeechResults{Error: errorx.IllegalState.Wrap(err, "failed to set speech voice to %q", conf.AzureVoice)}
		}
	}

	synthesizer, err := speech.NewSpeechSynthesizerFromConfig(speechConf, nil)
	if err != nil {
		return TextToSpeechResults{Error: errorx.IllegalState.Wrap(err, "failed to construct synthesizer")}
	}
	defer speechConf.Close()

	results := TextToSpeechResults{TextToSpeech: make(map[string]TextToSpeechResult, len(texts))}
	i := 0
	for text := range texts {
		i++ // increment counter at the beginning of the loop to make indices one-based instead of zero-based
		writeLog := func(msg string, args ...interface{}) {
			log.Printf("Speech synthesis [%d / %d]: "+msg, append([]interface{}{i, len(texts)}, args...)...)
		}

		writeLog("Start")
		audioData, err := func() ([]byte, error) {
			resultChan := synthesizer.SpeakTextAsync(text)
			var result speech.SpeechSynthesisOutcome
			select {
			case <-time.After(conf.AzureSynthesisTimeout):
				return nil, errorx.TimeoutElapsed.New("timeout elapsed for speech generation of text %s")
			case result = <-resultChan:
				// nop -- continue execution
			}
			if result.Error != nil {
				return nil, errorx.ExternalError.Wrap(err, "Speech synthesis failed")
			}
			defer result.Close()

			return result.Result.AudioData, nil
		}()

		if err != nil {
			writeLog("Failed with error %+v", err)
			results.TextToSpeech[text] = TextToSpeechResult{Error: err}
			continue
		}
		writeLog("Succeeded")
		results.TextToSpeech[text] = TextToSpeechResult{AudioData: audioData}
	}
	return results
}
