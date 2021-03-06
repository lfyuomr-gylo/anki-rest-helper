package ankihelper

import (
	"anki-rest-enhancer/ankiconnect"
	"anki-rest-enhancer/ankihelperconf"
	"anki-rest-enhancer/azuretts"
	"anki-rest-enhancer/util/stringx"
	"fmt"
	"github.com/joomcode/errorx"
	"log"
)

func NewHelper(
	ankiConnect ankiconnect.API,
	azureTTS azuretts.API,
) *Helper {
	return &Helper{
		ankiConnect: ankiConnect,
		azureTTS:    azureTTS,
	}
}

type Helper struct {
	ankiConnect ankiconnect.API
	azureTTS    azuretts.API
}

func (h Helper) Run(conf ankihelperconf.Actions) error {
	if err := h.ensureNoteTypes(conf.NoteTypes); err != nil {
		return err
	}
	if err := h.generateTTS(conf); err != nil {
		return err
	}
	if err := h.organizeCards(conf.CardsOrganization); err != nil {
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
	TextPreprocessors                 []ankihelperconf.TextProcessor
}

func (h Helper) generateTTS(conf ankihelperconf.Actions) error {
	log.Println("Generate test-to-speech...")

	// 0. Determine how to look for notes with missing Audio
	taskSources, err := h.getTTSTaskSources(conf)
	if err != nil {
		return err
	}

	// 1. Find all the notes with missing audio in Anki
	ttsTasks, err := h.findTTSTasks(taskSources)
	if err != nil {
		return err
	}
	if len(ttsTasks) == 0 {
		log.Println("No text to generate speech found. Skip text-to-speech generation")
		return nil
	}

	// 2. Generate audio for all the texts
	texts := make(map[string]struct{}, len(ttsTasks))
	for task := range ttsTasks {
		texts[task.Text] = struct{}{}
	}
	textToSpeech := h.azureTTS.TextToSpeech(texts)

	// 3. Update Anki Cards
	var succeeded, failed int
	for task := range ttsTasks {
		speech := textToSpeech[task.Text]
		if err := speech.Error; err != nil {
			log.Printf("Skip field %q in note %d due to text-to-speech error: %+v", task.TargetFieldName, task.NoteID, err)
			failed++
			continue
		}
		err := h.ankiConnect.UpdateNoteFields(task.NoteID, map[string]ankiconnect.FieldUpdate{
			task.TargetFieldName: {AudioData: speech.AudioMP3},
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

func (h Helper) getTTSTaskSources(conf ankihelperconf.Actions) ([]ttsTaskSource, error) {
	noteTypeByName := map[string]ankihelperconf.AnkiNoteType{}
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
				names := h.fieldNames(field)
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

func (h Helper) findTTSTasks(taskSources []ttsTaskSource) (map[ttsTask]struct{}, error) {
	ttsTasks := make(map[ttsTask]struct{})
	for i, tts := range taskSources {
		noteIDs, err := h.ankiConnect.FindNotes(tts.NoteFilter)
		if err != nil {
			return nil, errorx.Decorate(err, "failed to find notes matching filter for TTS #%d", i)
		}

		notes, err := h.ankiConnect.NotesInfo(noteIDs)
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

			task := ttsTask{
				NoteID:          noteID,
				Text:            text,
				TargetFieldName: tts.AudioField,
			}
			ttsTasks[task] = struct{}{}
		}
	}
	return ttsTasks, nil
}

func (h Helper) ensureNoteTypes(noteTypes []ankihelperconf.AnkiNoteType) error {
	if len(noteTypes) == 0 {
		log.Println("No note types defined in the configuration. Skip note type creation.")
		return nil
	}

	log.Println("Ensure note types...")
	existingNoteTypeNamesSlice, err := h.ankiConnect.ModelNames()
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

		if err := h.createNoteType(noteType); err != nil {
			return errorx.Decorate(err, "failed to create type type %q", noteType.Name)
		}
		created++
	}

	log.Printf("Finished Note Type creation (created/skipped): %d/%d", created, skipped)
	return nil
}

func (h Helper) createNoteType(conf ankihelperconf.AnkiNoteType) error {
	// Generate field names. First, we add Field, FieldExample and FieldExplanation
	// Voiceover fields are added at the end of the field list since they are not intended for manual modification
	var fieldNames []string
	var voiceoverFields []string
	for _, field := range conf.Fields {
		names := h.fieldNames(field)
		fieldNames = stringx.AppendNonEmpty(fieldNames, names.Field, names.Example, names.ExampleExplanation)
		voiceoverFields = stringx.AppendNonEmpty(voiceoverFields, names.FieldVoiceover, names.ExampleVoiceover)
	}
	fieldNames = append(fieldNames, voiceoverFields...)

	templates := make([]ankiconnect.CreateModelCardTemplate, 0, len(conf.Templates))
	for _, template := range conf.Templates {
		for _, field := range template.ForFields {
			names := h.fieldNames(field)
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
			if err := ankihelperconf.ValidateName(templateName); err != nil {
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

	return h.ankiConnect.CreateModel(params)
}

type FieldNames struct {
	Field, FieldVoiceover, Example, ExampleVoiceover, ExampleExplanation string
}

func (h Helper) fieldNames(field ankihelperconf.AnkiNoteField) FieldNames {
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

func (h Helper) organizeCards(rules []ankihelperconf.NotesOrganizationRule) error {
	log.Println("Applying notes organization rules...")
	for i, rule := range rules {
		log.Printf("Applying %d-th notes organization rule...", i)
		if err := h.applyOrganizationRule(rule); err != nil {
			return errorx.Decorate(err, "failed to apply %d-th notes organization rule", i)
		}
	}
	log.Println("Successfully applied notes organization rules.")
	return nil
}

func (h Helper) applyOrganizationRule(rule ankihelperconf.NotesOrganizationRule) error {
	targetDeck := rule.TargetDeckName
	query := fmt.Sprintf(`-"deck:%s" %s`, targetDeck, rule.NotesFilter)
	cardIDs, err := h.ankiConnect.FindCards(query)
	if err != nil {
		return err
	}
	if len(cardIDs) == 0 {
		log.Printf("Found no cards to be moved to deck %s.", targetDeck)
		return nil
	}
	log.Printf("Found %d cards to be moved to deck %s", len(cardIDs), targetDeck)

	if err := h.ankiConnect.ChangeDeck(targetDeck, cardIDs); err != nil {
		return err
	}
	log.Printf("Successfully moved %d cards to deck %s", len(cardIDs), targetDeck)
	return nil
}
