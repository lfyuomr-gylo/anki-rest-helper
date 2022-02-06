package enhance

import (
	"anki-rest-enhancer/ankiconnect"
	"anki-rest-enhancer/azuretts"
	"anki-rest-enhancer/enhancerconf"
	"anki-rest-enhancer/util/stringx"
	"fmt"
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
	if err := e.ensureNoteTypes(conf.Anki.NoteTypes); err != nil {
		return err
	}
	if err := e.generateTTS(conf); err != nil {
		return err
	}

	return nil
}

type ttsTask struct {
	NoteID          ankiconnect.NoteID
	Text            string
	TargetFieldName string
}

type ttsTaskSource struct {
	NoteFilter, TextField, AudioField string
	TextPreprocessors                 []enhancerconf.TextProcessor
}

func (e Enhancer) generateTTS(conf enhancerconf.Config) error {
	log.Println("Generate test-to-speech...")

	// 0. Determine how to look for notes with missing Audio
	taskSources, err := e.getTTSTaskSources(conf.Anki)
	if err != nil {
		return err
	}

	// 1. Find all the notes with missing audio in Anki
	ttsTasks, err := e.findTTSTasks(taskSources)
	if err != nil {
		return err
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

func (e Enhancer) getTTSTaskSources(conf enhancerconf.Anki) ([]ttsTaskSource, error) {
	noteTypeByName := map[string]enhancerconf.AnkiNoteType{}
	for _, noteType := range conf.NoteTypes {
		noteTypeByName[noteType.Name] = noteType
	}

	// 0. Determine note filters to use
	var taskSources []ttsTaskSource
	for i, tts := range conf.TTS {
		switch {
		case tts.Fields != nil:
			taskSources = append(taskSources, ttsTaskSource{
				NoteFilter:        tts.Fields.NoteFilter,
				TextField:         tts.Fields.TextField,
				AudioField:        tts.Fields.AudioField,
				TextPreprocessors: tts.TextPreprocessors,
			})
		case tts.GeneratedNoteTypeName != nil:
			typeName := *tts.GeneratedNoteTypeName
			noteType, ok := noteTypeByName[typeName]
			if !ok {
				return nil, errorx.IllegalState.New("Broken generated note type reference %q in TTS #%d", typeName, i)
			}
			for _, field := range noteType.Fields {
				names := e.fieldNames(field)
				if names.Field != "" && names.FieldVoiceover != "" {
					taskSources = append(taskSources, ttsTaskSource{
						NoteFilter:        fmt.Sprintf(`"note:%s" "%s:_*" "%s:"`, typeName, names.Field, names.FieldVoiceover),
						TextField:         names.Field,
						AudioField:        names.FieldVoiceover,
						TextPreprocessors: tts.TextPreprocessors,
					})
				}
				if names.Example != "" && names.ExampleVoiceover != "" {
					taskSources = append(taskSources, ttsTaskSource{
						NoteFilter:        fmt.Sprintf(`"note:%s" "%s:_*" "%s:"`, typeName, names.Example, names.ExampleVoiceover),
						TextField:         names.Example,
						AudioField:        names.ExampleVoiceover,
						TextPreprocessors: tts.TextPreprocessors,
					})
				}
			}
		default:
			panic(errorx.Panic(errorx.IllegalState.New("unexpected tts: %+v", tts)))
		}
	}
	return taskSources, nil
}

func (e Enhancer) findTTSTasks(taskSources []ttsTaskSource) ([]ttsTask, error) {
	var ttsTasks []ttsTask
	for i, tts := range taskSources {
		noteIDs, err := e.ankiConnect.FindNotes(tts.NoteFilter)
		if err != nil {
			return nil, errorx.Decorate(err, "failed to find notes matching filter for TTS #%d", i)
		}

		notes, err := e.ankiConnect.NotesInfo(noteIDs)
		if err != nil {
			return nil, errorx.Decorate(err, "failed to obtain notes matching filter for TTS #%d", i)
		}

		for noteID, note := range notes {
			text, ok := note.Fields[tts.TextField]
			if !ok {
				return nil, errorx.IllegalState.New("There is no field %q in note %d", tts.TextField, noteID)
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
	return ttsTasks, nil
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
			for name, val := range field.Vars {
				if _, ok := substitutions[name]; ok {
					return errorx.IllegalState.New("custom variable %q collides with a default variable in template %q", name, template.Name)
				}
				substitutions[name] = val
			}
			stringx.RemoveEmptyValues(substitutions)

			templateName, err := substituteVariables(template.Name, substitutions)
			if err != nil {
				return errorx.Decorate(err, "failed to build card template name for template %q and field %q", template.Name, field.Name)
			}
			if err := enhancerconf.ValidateName(templateName); err != nil {
				return errorx.Decorate(err, "got invalid template name after variables substitution: %s", templateName)
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
				Front: fmt.Sprintf("{{#%s}}\n%s\n{{/%s}}", field.Name, front, field.Name),
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
