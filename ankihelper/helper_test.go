package ankihelper_test

import (
	"anki-rest-enhancer/ankiconnect"
	"anki-rest-enhancer/ankiconnect/ankiconnectmock"
	"anki-rest-enhancer/ankihelper"
	"anki-rest-enhancer/ankihelperconf"
	"anki-rest-enhancer/azuretts"
	"anki-rest-enhancer/azuretts/azurettsmock"
	"github.com/stretchr/testify/suite"
	"testing"
)

func TestEnhancer(t *testing.T) {
	suite.Run(t, &EnhancerSuite{})
}

type EnhancerSuite struct {
	suite.Suite

	Enhancer *ankihelper.Helper
	TTSMock  *azurettsmock.API
	AnkiMock *ankiconnectmock.API
}

func (s *EnhancerSuite) SetupSuite() {
	s.TTSMock = &azurettsmock.API{}
	s.AnkiMock = &ankiconnectmock.API{}
	s.Enhancer = ankihelper.NewHelper(s.AnkiMock, s.TTSMock)
}

func (s *EnhancerSuite) SetupTest() {
	s.TTSMock.Reset()
	s.AnkiMock.Reset()
}

func (s *EnhancerSuite) TestNoteTypeCreation_AlreadyExists() {
	// setup:
	const modelName = "my model"
	s.AnkiMock.ModelNamesFunc = func() ([]string, error) {
		return []string{modelName}, nil
	}
	var createModelCalls []ankiconnect.CreateModelParams
	s.AnkiMock.CreateModelFunc = func(params ankiconnect.CreateModelParams) error {
		createModelCalls = append(createModelCalls, params)
		return nil
	}

	// given:
	actions := ankihelperconf.Actions{NoteTypes: []ankihelperconf.AnkiNoteType{{
		Name:      modelName,
		Fields:    []ankihelperconf.AnkiNoteField{{Name: "Foo"}, {Name: "Bar"}},
		Templates: []ankihelperconf.AnkiCardTemplate{{Name: "Card1", ForFields: []ankihelperconf.AnkiNoteField{{Name: "foo"}}}},
	}}}

	// when:
	err := s.Enhancer.Run(actions)

	// then:
	s.Require().NoError(err)
	s.Require().Empty(createModelCalls, "Helper should not attempt to recreate already existing note types")
}

func (s *EnhancerSuite) TestNoteTypeCreation_CreateNewWithExampleAndVoiceover() {
	// setup:
	s.AnkiMock.ModelNamesFunc = func() ([]string, error) {
		return []string{"existing model"}, nil
	}
	var createModelCalls []ankiconnect.CreateModelParams
	s.AnkiMock.CreateModelFunc = func(params ankiconnect.CreateModelParams) error {
		createModelCalls = append(createModelCalls, params)
		return nil
	}

	// given:
	fieldName := ankihelperconf.AnkiNoteField{
		Name: "word",
		Vars: map[string]string{"TITLE": "Word", "MY_EMPTY_FIELD": ""},
	}
	actions := ankihelperconf.Actions{NoteTypes: []ankihelperconf.AnkiNoteType{{
		Name:   "My Note Type",
		CSS:    ".foo { font-size: large; }",
		Fields: []ankihelperconf.AnkiNoteField{fieldName},
		Templates: []ankihelperconf.AnkiCardTemplate{{
			Name:      "WordTemplate",
			ForFields: []ankihelperconf.AnkiNoteField{fieldName},
			Front:     "$TITLE$: {{ $FIELD$ }}\nExample: {{ $EXAMPLE$ }}\nExplanation: {{ $EXAMPLE_EXPLANATION$ }}$MY_EMPTY_FIELD$",
			Back:      "{{ $FIELD_VOICEOVER$ }} {{ $EXAMPLE_VOICEOVER$ }}",
		}},
	}}}
	expectedModel := ankiconnect.CreateModelParams{
		ModelName:     "My Note Type",
		InOrderFields: []string{"word", "wordExample", "wordExampleExplanation", "wordVoiceover", "wordExampleVoiceover"},
		CSS:           ".foo { font-size: large; }",
		IsCloze:       false,
		CardTemplates: []ankiconnect.CreateModelCardTemplate{{
			Name:  "WordTemplate",
			Front: "{{#word}}\nWord: {{ word }}\nExample: {{ wordExample }}\nExplanation: {{ wordExampleExplanation }}\n{{/word}}",
			Back:  "{{ wordVoiceover }} {{ wordExampleVoiceover }}",
		}},
	}

	// when:
	err := s.Enhancer.Run(actions)

	// then:
	s.Require().NoError(err)
	s.Require().Len(createModelCalls, 1)
	s.Require().Equal(expectedModel, createModelCalls[0])
}

func (s *EnhancerSuite) TestNoteTypeCreation_CreateNewWithNoExampleOrVoiceover() {
	// setup:
	s.AnkiMock.ModelNamesFunc = func() ([]string, error) {
		return []string{"existing model"}, nil
	}
	var createModelCalls []ankiconnect.CreateModelParams
	s.AnkiMock.CreateModelFunc = func(params ankiconnect.CreateModelParams) error {
		createModelCalls = append(createModelCalls, params)
		return nil
	}

	// given:
	fieldComment := ankihelperconf.AnkiNoteField{
		Name:          "comment",
		SkipExample:   true,
		SkipVoiceover: true,
	}
	const css = ".foo { font-size: large; }"
	actions := ankihelperconf.Actions{NoteTypes: []ankihelperconf.AnkiNoteType{{
		Name:   "MyModel",
		CSS:    css,
		Fields: []ankihelperconf.AnkiNoteField{fieldComment},
		Templates: []ankihelperconf.AnkiCardTemplate{{
			Name:      "CommentTemplate",
			ForFields: []ankihelperconf.AnkiNoteField{fieldComment},
			Front:     "Field Name: $FIELD$",
			Back:      "Field Conent: {{ $FIELD$ }}",
		}},
	}}}
	expectedModel := ankiconnect.CreateModelParams{
		ModelName:     "MyModel",
		InOrderFields: []string{"comment"},
		CSS:           css,
		IsCloze:       false,
		CardTemplates: []ankiconnect.CreateModelCardTemplate{{
			Name:  "CommentTemplate",
			Front: "{{#comment}}\nField Name: comment\n{{/comment}}",
			Back:  "Field Conent: {{ comment }}",
		}},
	}

	// when:
	err := s.Enhancer.Run(actions)

	// then:
	s.Require().NoError(err)
	s.Require().Len(createModelCalls, 1)
	s.Require().Equal(expectedModel, createModelCalls[0])
}

func (s *EnhancerSuite) TestTTSGeneration_Simple() {
	// given:
	const (
		query                                    = "foo:_* fooVoiceover:"
		noteID                ankiconnect.NoteID = 42
		textField, audioField                    = "foo", "bar"
		text, audio                              = "¡Hola, buenos días!", "abacabadabacaba"
	)
	actions := ankihelperconf.Actions{
		TTS: []ankihelperconf.AnkiTTS{{
			Fields: &ankihelperconf.AnkiTTSFields{
				NoteFilter: query,
				TextField:  textField,
				AudioField: audioField,
			},
		}},
		NoteTypes: nil,
	}
	expectedNoteUpdate := map[string]ankiconnect.FieldUpdate{audioField: {AudioData: []byte(audio)}}

	// setup:
	s.AnkiMock.FindNotesFunc = func(aQuery string) ([]ankiconnect.NoteID, error) {
		s.Require().Equal(query, aQuery)
		return []ankiconnect.NoteID{noteID}, nil
	}
	s.AnkiMock.NotesInfoFunc = func(noteIDs []ankiconnect.NoteID) (map[ankiconnect.NoteID]ankiconnect.NoteInfo, error) {
		s.Require().Equal([]ankiconnect.NoteID{noteID}, noteIDs)
		return map[ankiconnect.NoteID]ankiconnect.NoteInfo{
			noteID: {Fields: map[string]string{
				textField:  text,
				audioField: "",
			}},
		}, nil
	}
	s.TTSMock.TextToSpeechFunc = func(texts map[string]struct{}) map[string]azuretts.TextToSpeechResult {
		s.Require().Equal(map[string]struct{}{text: {}}, texts)
		return map[string]azuretts.TextToSpeechResult{text: {AudioMP3: []byte(audio)}}
	}
	var updatedFields map[string]ankiconnect.FieldUpdate
	s.AnkiMock.UpdateNoteFieldsFunc = func(aNoteID ankiconnect.NoteID, fields map[string]ankiconnect.FieldUpdate) error {
		s.Require().Equal(noteID, aNoteID)
		s.Require().Nil(updatedFields, "notes should be updated at most once")
		updatedFields = fields
		return nil
	}

	// when:
	err := s.Enhancer.Run(actions)

	// then:
	s.Require().NoError(err)
	s.Require().Equal(expectedNoteUpdate, updatedFields)
}

func (s *EnhancerSuite) TestTTSGeneration_SingleErrorIsIgnored() {
	// given: note1
	const (
		textField, audioField                    = "text", "audio"
		noteID1, noteID2      ankiconnect.NoteID = 42, 16
		query                                    = "text:_* audio:"
		text1                                    = "¿Qué pasa?"
		text2, audio2                            = "Ahora yo escribo el test", "abacabadabacaba"
	)
	actions := ankihelperconf.Actions{
		TTS: []ankihelperconf.AnkiTTS{{
			Fields: &ankihelperconf.AnkiTTSFields{
				NoteFilter: query,
				TextField:  textField,
				AudioField: audioField,
			},
		}},
		NoteTypes: nil,
	}
	// Speech generation fails for the first note, so we expect the enhancer to skip that note and only
	// update the second note for which the generation succeeded.
	type noteUpdatesMap = map[ankiconnect.NoteID]map[string]ankiconnect.FieldUpdate
	expectedUpdates := noteUpdatesMap{noteID2: {audioField: {AudioData: []byte(audio2)}}}

	// setup:
	s.AnkiMock.FindNotesFunc = func(aQuery string) ([]ankiconnect.NoteID, error) {
		s.Require().Equal(query, aQuery)
		return []ankiconnect.NoteID{noteID1, noteID2}, nil
	}
	s.AnkiMock.NotesInfoFunc = func(noteIDs []ankiconnect.NoteID) (map[ankiconnect.NoteID]ankiconnect.NoteInfo, error) {
		s.Require().ElementsMatch([]ankiconnect.NoteID{noteID1, noteID2}, noteIDs)
		return map[ankiconnect.NoteID]ankiconnect.NoteInfo{
			noteID1: {Fields: map[string]string{textField: text1, audioField: ""}},
			noteID2: {Fields: map[string]string{textField: text2, audioField: ""}},
		}, nil
	}
	s.TTSMock.TextToSpeechFunc = func(texts map[string]struct{}) map[string]azuretts.TextToSpeechResult {
		s.Require().Equal(map[string]struct{}{text1: {}, text2: {}}, texts)
		return map[string]azuretts.TextToSpeechResult{
			text1: {Error: azuretts.TooManyRequests.NewWithNoMessage()},
			text2: {AudioMP3: []byte(audio2)},
		}
	}
	noteUpdates := make(noteUpdatesMap)
	s.AnkiMock.UpdateNoteFieldsFunc = func(noteID ankiconnect.NoteID, fields map[string]ankiconnect.FieldUpdate) error {
		s.Require().NotContains(noteUpdates, noteID, "note has already been updated")
		noteUpdates[noteID] = fields
		return nil
	}

	// when:
	err := s.Enhancer.Run(actions)

	// then:
	s.Require().NoError(err)
	s.Require().Equal(expectedUpdates, noteUpdates)
}
