package enhance

import (
	"anki-rest-enhancer/ankiconnect"
	"anki-rest-enhancer/azuretts"
	"anki-rest-enhancer/enhancerconf"
	"github.com/joomcode/errorx"
	"log"
)

func NewEnhancer(conf enhancerconf.Config) *Enhancer {
	return &Enhancer{
		ankiConnect: ankiconnect.NewAPI(conf.Anki),
		azureTTS:    azuretts.NewAPI(conf.Azure),
	}
}

type Enhancer struct {
	ankiConnect *ankiconnect.API
	azureTTS    *azuretts.API
}

func (e Enhancer) Enhance(conf enhancerconf.Config) error {
	return e.generateTTS(conf)
}

type ttsTask struct {
	NoteID          ankiconnect.NoteID
	Text            string
	TargetFieldName string
}

func (e Enhancer) generateTTS(conf enhancerconf.Config) error {
	// 1. Find all the notes with missing audio in Anki
	var ttsTasks []ttsTask
	for i, tts := range conf.Anki.TTS {
		noteIDs, err := e.ankiConnect.FindNotes(tts.NoteFilter)
		if err != nil {
			return errorx.Decorate(err, "failed to find notes matching filter for TTS #%d", i)
		}

		notes, err := e.ankiConnect.NotesInfo(noteIDs)
		if err != nil {
			return errorx.Decorate(err, "failed to obtain notes matching filter for TTS #%d", i)
		}

		for noteID, note := range notes {
			text, ok := note.Fields[tts.TextField]
			if !ok {
				return errorx.IllegalState.New("There is no field %q in note %d", tts.TextField, noteID)
			}
			for _, preprocessor := range tts.TextPreprocessors {
				text = preprocessor.Process(text)
			}

			ttsTasks = append(ttsTasks, ttsTask{
				NoteID:          noteID,
				Text:            text,
				TargetFieldName: tts.AudioField,
			})
		}
	}

	// 2. Generate audio for all the texts
	texts := make(map[string]struct{}, len(ttsTasks))
	for _, task := range ttsTasks {
		texts[task.Text] = struct{}{}
	}
	textToSpeech := e.azureTTS.TextToSpeech(texts)

	// 3. Update Anki Cards
	var succeeded, failed int
	for _, task := range ttsTasks {
		speech := textToSpeech[task.Text]
		if err := speech.Error; err != nil {
			log.Printf("Skip field %q in note %d due to text-to-speech error: %+v", task.TargetFieldName, task.NoteID, err)
			failed++
			continue
		}
		err := e.ankiConnect.UpdateNoteFields(task.NoteID, map[string]ankiconnect.FieldUpdate{
			task.TargetFieldName: {AudioData: speech.AudioData},
		})
		if err != nil {
			log.Printf("Failed to update field %q of note %d due to AnkiConnect error: %+v", task.TargetFieldName, task.NoteID, err)
			failed++
			continue
		}
		succeeded++
	}

	log.Printf("Finished text-to-speech generation. Generations count (succeeded/failed): %d/%d", succeeded, failed)
	return nil
}
