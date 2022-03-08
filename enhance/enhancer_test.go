package enhance_test

import (
	"anki-rest-enhancer/ankiconnect"
	"anki-rest-enhancer/ankiconnect/ankiconnectmock"
	"anki-rest-enhancer/azuretts"
	"anki-rest-enhancer/azuretts/azurettsmock"
	"anki-rest-enhancer/enhance"
	"anki-rest-enhancer/enhancerconf"
	"github.com/stretchr/testify/suite"
	"testing"
)

func TestEnhancer(t *testing.T) {
	suite.Run(t, &EnhancerSuite{})
}

type EnhancerSuite struct {
	suite.Suite

	Enhancer *enhance.Enhancer
	TTSMock  *azurettsmock.API
	AnkiMock *ankiconnectmock.API
}

func (s *EnhancerSuite) SetupSuite() {
	s.TTSMock = &azurettsmock.API{}
	s.AnkiMock = &ankiconnectmock.API{}
	s.Enhancer = enhance.NewEnhancer(s.AnkiMock, s.TTSMock)
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
	actions := enhancerconf.Actions{NoteTypes: []enhancerconf.AnkiNoteType{{
		Name:      modelName,
		Fields:    []enhancerconf.AnkiNoteField{{Name: "Foo"}, {Name: "Bar"}},
		Templates: []enhancerconf.AnkiCardTemplate{{Name: "Card1", ForFields: []enhancerconf.AnkiNoteField{{Name: "foo"}}}},
	}}}

	// when:
	err := s.Enhancer.Enhance(actions)

	// then:
	s.Require().NoError(err)
	s.Require().Empty(createModelCalls, "Enhancer should not attempt to recreate already existing note types")
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
	fieldName := enhancerconf.AnkiNoteField{
		Name: "word",
		Vars: map[string]string{"TITLE": "Word"},
	}
	actions := enhancerconf.Actions{NoteTypes: []enhancerconf.AnkiNoteType{{
		Name:   "My Note Type",
		CSS:    ".foo { font-size: large; }",
		Fields: []enhancerconf.AnkiNoteField{fieldName},
		Templates: []enhancerconf.AnkiCardTemplate{{
			Name:      "WordTemplate",
			ForFields: []enhancerconf.AnkiNoteField{fieldName},
			Front:     "$TITLE$: {{ $FIELD$ }}\nExample: {{ $EXAMPLE$ }}\nExplanation: {{ $EXAMPLE_EXPLANATION$ }}",
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
	err := s.Enhancer.Enhance(actions)

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
	fieldComment := enhancerconf.AnkiNoteField{
		Name:          "comment",
		SkipExample:   true,
		SkipVoiceover: true,
	}
	const css = ".foo { font-size: large; }"
	actions := enhancerconf.Actions{NoteTypes: []enhancerconf.AnkiNoteType{{
		Name:   "MyModel",
		CSS:    css,
		Fields: []enhancerconf.AnkiNoteField{fieldComment},
		Templates: []enhancerconf.AnkiCardTemplate{{
			Name:      "CommentTemplate",
			ForFields: []enhancerconf.AnkiNoteField{fieldComment},
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
	err := s.Enhancer.Enhance(actions)

	// then:
	s.Require().NoError(err)
	s.Require().Len(createModelCalls, 1)
	s.Require().Equal(expectedModel, createModelCalls[0])
}

func (s *EnhancerSuite) TestTTSGeneration_Simple() {
	// given:
	const (
		query                         = "foo:_* fooVoiceover:"
		noteID     ankiconnect.NoteID = 42
		textField                     = "foo"
		audioField                    = "bar"
		text                          = "¡Hola, buenos días!"
		audio                         = "abacabadabacaba"
	)
	actions := enhancerconf.Actions{
		TTS: []enhancerconf.AnkiTTS{{
			Fields: &enhancerconf.AnkiTTSFields{
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
	err := s.Enhancer.Enhance(actions)

	// then:
	s.Require().NoError(err)
	s.Require().Equal(expectedNoteUpdate, updatedFields)
}
