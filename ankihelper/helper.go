package ankihelper

import (
	"anki-rest-enhancer/ankiconnect"
	"anki-rest-enhancer/ankihelperconf"
	"anki-rest-enhancer/azuretts"
	"anki-rest-enhancer/ratelimit"
	"anki-rest-enhancer/util/execx"
	"anki-rest-enhancer/util/iox"
	"anki-rest-enhancer/util/lang"
	"anki-rest-enhancer/util/stringx"
	"anki-rest-enhancer/util/templatex"
	"context"
	"encoding/json"
	"fmt"
	"github.com/joomcode/errorx"
	"log"
	"os"
	"strings"
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
	ctx := context.TODO()

	if err := h.uploadMedia(conf.UploadMedia); err != nil {
		return err
	}
	if err := h.ensureNoteTypes(conf.NoteTypes); err != nil {
		return err
	}
	if err := h.processNotes(ctx, conf.NoteProcessing); err != nil {
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

func (h Helper) uploadMedia(media []ankihelperconf.AnkiUploadMedia) error {
	for i, mediaUpload := range media {
		if err := h.uploadSingleMedia(mediaUpload); err != nil {
			return errorx.Decorate(err, "failed to upload media #%d", i)
		}
	}
	return nil
}

func (h Helper) uploadSingleMedia(media ankihelperconf.AnkiUploadMedia) error {
	f, err := os.Open(media.FilePath)
	if err != nil {
		return errorx.ExternalError.Wrap(err, "failed to open media file %q", media.FilePath)
	}
	defer iox.Close(f)

	log.Printf("Uploading file %q to Anki under name %q...", media.FilePath, media.AnkiName)
	return h.ankiConnect.StoreMediaFile(media.AnkiName, f, true)
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
		fieldNames = stringx.AppendNonEmpty(fieldNames, names.Field)
		voiceoverFields = stringx.AppendNonEmpty(voiceoverFields, names.FieldVoiceover)
	}
	fieldNames = append(fieldNames, voiceoverFields...)

	templates := make([]ankiconnect.CreateModelCardTemplate, 0, len(conf.Templates))
	for tmplIdx, cardTemplate := range conf.Templates {
		for _, field := range cardTemplate.ForFields {
			const voiceoverSuffix = "Voiceover"
			substitutions := map[string]any{
				"Field": field.Name,
				"Vars":  field.Vars,
			}
			if !field.SkipVoiceover {
				substitutions["FieldVoiceover"] = field.Name + voiceoverSuffix
			}

			cardTemplateName, err := templatex.Execute(cardTemplate.Name, substitutions)
			if err != nil {
				return errorx.Decorate(err, "failed to build card template name for template #%d and field %q", tmplIdx, field.Name)
			}
			if err := ankihelperconf.ValidateName(cardTemplateName); err != nil {
				return errorx.Decorate(err, "got invalid template name after variables substitution: %s", cardTemplateName)
			}
			front, err := templatex.Execute(cardTemplate.Front, substitutions)
			if err != nil {
				return errorx.Decorate(err, "failed to build card template front for template #%d and field %q", tmplIdx, field.Name)
			}
			back, err := templatex.Execute(cardTemplate.Back, substitutions)
			if err != nil {
				return errorx.Decorate(err, "failed to build card template back for template #%d and field %q", tmplIdx, field.Name)
			}

			templates = append(templates, ankiconnect.CreateModelCardTemplate{
				Name:  cardTemplateName,
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
	Field, FieldVoiceover string
}

func (h Helper) fieldNames(field ankihelperconf.AnkiNoteField) FieldNames {
	const voiceoverSuffix = "Voiceover"

	names := FieldNames{Field: field.Name}
	if !field.SkipVoiceover {
		names.FieldVoiceover = field.Name + voiceoverSuffix
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

func (h Helper) processNotes(ctx context.Context, rules []ankihelperconf.NoteProcessingRule) error {
	log.Println("Process notes...")

	for i, rule := range rules {
		log.Printf("Running note processing rule #%d...", i)
		if err := h.applyProcessingRule(ctx, rule); err != nil {
			return errorx.Decorate(err, "failed to execute note population rule #%d", i)
		}
	}

	log.Println("Successfully completed notes population with auto-generated content!")
	return nil
}

func (h Helper) applyProcessingRule(ctx context.Context, rule ankihelperconf.NoteProcessingRule) error {
	// 1. find notes to populate
	noteIDs, err := h.ankiConnect.FindNotes(rule.NoteFilter)
	if err != nil {
		return err
	}
	notes, err := h.ankiConnect.NotesInfo(noteIDs)
	if err != nil {
		return err
	}
	log.Printf("Found %d notes to process...", len(notes))

	// 2. for each note, run population
	throttler := ratelimit.NewThrottler(rule.MinPauseBetweenExecutions)
	idx := 0
	for noteID, note := range notes {
		idx++
		throttler.Throttle()

		err := h.processNote(ctx, rule, note, idx, len(notes))
		if err != nil {
			log.Printf("Failed to process note %d, error: %s", noteID, err)
			continue
		}
	}

	return nil
}

type noteProcessingModification struct {
	// oneof
	SetField           *map[string]string `json:"set_field"`
	SetFieldIfNotEmpty *map[string]string `json:"set_field_if_not_empty"`
	AddTag             *string            `json:"add_tag"`
}

func (m noteProcessingModification) Validate() error {
	fieldsSet := 0
	if m.SetField != nil {
		fieldsSet++
	}
	if m.AddTag != nil {
		fieldsSet++
	}
	if m.SetFieldIfNotEmpty != nil {
		fieldsSet++
	}

	if fieldsSet != 1 {
		return errorx.IllegalFormat.New("invalid note modification command has %d top-level keys instead of one: %+v", fieldsSet, m)
	}
	return nil
}

func (h Helper) processNote(
	ctx context.Context,
	rule ankihelperconf.NoteProcessingRule,
	note ankiconnect.NoteInfo,
	noteIdx, totalNotes int,
) error {
	templateContext := map[string]any{
		"Note": map[string]any{
			"Fields": note.Fields,
			"Tags":   note.Tags,
		},
	}

	args := make([]string, len(rule.Exec.Args))
	for i, arg := range rule.Exec.Args {
		switch {
		case arg.PlainString != nil:
			args[i] = *arg.PlainString
		case arg.Template != nil:
			var argBuilder strings.Builder
			if err := arg.Template.Execute(&argBuilder, templateContext); err != nil {
				return errorx.IllegalFormat.Wrap(err, "failed to substitute template in argument #%d", i)
			}
			args[i] = argBuilder.String()
		}
	}

	var stdin string
	switch {
	case rule.Exec.Stdin.PlainString != nil:
		stdin = *rule.Exec.Stdin.PlainString
	case rule.Exec.Stdin.Template != nil:
		var stdinBuilder strings.Builder
		if err := rule.Exec.Stdin.Template.Execute(&stdinBuilder, templateContext); err != nil {
			return errorx.IllegalFormat.Wrap(err, "failed to substitute template in stdin of the script")
		}
		stdin = stdinBuilder.String()
	}

	cmdCtx := ctx
	if rule.Timeout > 0 {
		// currently this context deadline is not respected by
		ctx, cancel := context.WithTimeout(cmdCtx, rule.Timeout)
		defer cancel()
		cmdCtx = ctx
	}
	log.Printf("Executing note processing command [%d/%d]: %s %s", noteIdx, totalNotes, rule.Exec.Command, strings.Join(args, " "))
	cmdOut, err := execx.RunAndCollectOutput(cmdCtx, execx.Params{
		Command: rule.Exec.Command,
		Args:    args,
		Stdin:   stdin,
	})
	if err != nil {
		return errorx.ExternalError.Wrap(err, "Note population command failed")
	}

	var commandOutParsed []noteProcessingModification
	if !stringx.IsBlank(string(cmdOut)) {
		if err := json.Unmarshal(cmdOut, &commandOutParsed); err != nil {
			return errorx.ExternalError.Wrap(err, "Note processing command's stdout is malformed")
		}
		for idx, modification := range commandOutParsed {
			if err := modification.Validate(); err != nil {
				return errorx.Decorate(err, "Note processing command's stdout contains malformed modification #%d", idx)
			}
		}
	}

	fieldUpdates := make(map[string]ankiconnect.FieldUpdate)
	var tagsToAdd []string
	for _, modification := range commandOutParsed {
		switch {
		case modification.SetField != nil:
			for field, value := range *modification.SetField {
				fieldUpdates[field] = ankiconnect.FieldUpdate{Value: lang.New(value)}
			}
		case modification.SetFieldIfNotEmpty != nil:
			for field, value := range *modification.SetFieldIfNotEmpty {
				if note.Fields[field] == "" {
					fieldUpdates[field] = ankiconnect.FieldUpdate{Value: lang.New(value)}
				}
			}
		case modification.AddTag != nil:
			tagsToAdd = append(tagsToAdd, *modification.AddTag)
		default:
			panic(errorx.IllegalState.New("Unexpected modification: %+v", modification))
		}
	}

	if len(fieldUpdates) > 0 {
		if err := h.ankiConnect.UpdateNoteFields(note.ID, fieldUpdates); err != nil {
			return err
		}
	}
	if len(tagsToAdd) > 0 {
		if err := h.ankiConnect.AddTags([]ankiconnect.NoteID{note.ID}, tagsToAdd); err != nil {
			return err
		}
	}
	return nil
}
