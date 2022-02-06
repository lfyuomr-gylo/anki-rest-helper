package enhance

import (
	"anki-rest-enhancer/ankiconnect"
	"anki-rest-enhancer/azuretts"
	"anki-rest-enhancer/enhancerconf"
	"anki-rest-enhancer/util/stringx"
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
	if err := e.generateTTS(conf); err != nil {
		return err
	}
	if err := e.ensureNoteTypes(conf.Anki.NoteTypes); err != nil {
		return err
	}

	return nil
}

type ttsTask struct {
	NoteID          ankiconnect.NoteID
	Text            string
	TargetFieldName string
}

func (e Enhancer) generateTTS(conf enhancerconf.Config) error {
	log.Println("Generate test-to-speech...")
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

func (e Enhancer) ensureNoteTypes(noteTypes []enhancerconf.AnkiNoteType) error {
	log.Println("Ensure note types...")
	existingNoteTypeNamesSlice, err := e.ankiConnect.ModelNames()
	if err != nil {
		return err
	}
	existingNoteTypeNamesSet := make(map[string]struct{}, len(existingNoteTypeNamesSlice))
	for _, name := range existingNoteTypeNamesSlice {
		existingNoteTypeNamesSet[name] = struct{}{}
	}

	var created, skipped int
	for _, noteType := range noteTypes {
		if _, ok := existingNoteTypeNamesSet[noteType.Name]; ok {
			log.Printf("Note Type %q already exists in Anki. Skip its creation...", noteType.Name)
			skipped++
			continue
		}

		if err := e.createNoteType(noteType); err != nil {
			return errorx.Decorate(err, "failed to create type type %q", noteType.Name)
		}
		created++
	}

	log.Printf("Finished Note Type creation (created/skipped): %d/%d", created, skipped)
	return nil
}

func (e Enhancer) createNoteType(conf enhancerconf.AnkiNoteType) error {
	// Generate field names. First, we add Field, FieldExample and FieldExplanation
	// Voiceover fields are added at the end of the field list since they are not intended for manual modification
	var fieldNames []string
	var voiceoverFields []string
	for _, field := range conf.Fields {
		names := e.fieldNames(field)
		fieldNames = stringx.AppendNonEmpty(fieldNames, names.Field, names.Example, names.ExampleExplanation)
		voiceoverFields = stringx.AppendNonEmpty(voiceoverFields, names.FieldVoiceover, names.ExampleVoiceover)
	}
	fieldNames = append(fieldNames, voiceoverFields...)

	templates := make([]ankiconnect.CreateModelCardTemplate, 0, len(conf.Templates))
	for _, template := range conf.Templates {
		for _, field := range template.ForFields {
			names := e.fieldNames(field)
			substitutions := map[string]string{
				"FIELD":               names.Field,
				"FIELD_VOICEOVER":     names.FieldVoiceover,
				"EXAMPLE":             names.Example,
				"EXAMPLE_VOICEOVER":   names.ExampleVoiceover,
				"EXAMPLE_EXPLANATION": names.ExampleExplanation,
			}
			stringx.RemoveEmptyValues(substitutions)

			templateName, err := substituteVariables(template.Name, substitutions)
			if err != nil {
				return errorx.Decorate(err, "failed to build card template name for template %q and field %q", template.Name, field.Name)
			}
			front, err := substituteVariables(template.Front, substitutions)
			if err != nil {
				return errorx.Decorate(err, "failed to build card template front for %q and field %q", template.Name, field.Name)
			}
			back, err := substituteVariables(template.Back, substitutions)
			if err != nil {
				return errorx.Decorate(err, "failed to build card template back for %q and field %q", template.Name, field.Name)
			}

			templates = append(templates, ankiconnect.CreateModelCardTemplate{
				Name:  templateName,
				Front: front,
				Back:  back,
			})
		}
	}

	params := ankiconnect.CreateModelParams{
		ModelName:     conf.Name,
		InOrderFields: fieldNames,
		CSS:           conf.CSS,
		IsCloze:       false,
		CardTemplates: templates,
	}

	return e.ankiConnect.CreateModel(params)
}

type FieldNames struct {
	Field, FieldVoiceover, Example, ExampleVoiceover, ExampleExplanation string
}

func (e Enhancer) fieldNames(field enhancerconf.AnkiNoteField) FieldNames {
	const voiceoverSuffix = "Voiceover"
	const exampleSuffix = "Example"
	const explanationSuffix = "Explanation"

	names := FieldNames{Field: field.Name}
	if !field.SkipVoiceover {
		names.FieldVoiceover = field.Name + voiceoverSuffix
	}
	if !field.SkipExample {
		names.Example = field.Name + exampleSuffix
		names.ExampleExplanation = names.Example + explanationSuffix
		if !field.SkipVoiceover {
			names.ExampleVoiceover = names.Example + voiceoverSuffix
		}
	}

	return names
}
