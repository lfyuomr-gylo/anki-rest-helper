// Usage:
//   go run ./german_word_forms.go 'word type' 'word' '["tag1", "tag2"]'

package main

import (
	"anki-rest-enhancer/util/lang/mapx"
	"anki-rest-enhancer/util/lang/set"
	"anki-rest-enhancer/util/lang/slicex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
)

const (
	entryCodeNoun = "nn"
	entryCodeVerb = "vrb"

	// special value to indicate that the target row of the conjugation table
	// contains only one entry, and that this entry should be taken as is.
	pronounTakeOnlyEntry = "$TAKE_ONLY_ENTRY$"
)

var (
	ErrNoDictEntries       = errors.New("no dict entries")
	ErrMultipleDictEntries = errors.New("too many dict entries")
)

type ankiCommand = map[string]any

type YandexLookupResult struct {
	// Entry in the 'de-ru' dictionary
	DeRu struct {
		Regular []YandexDictEntry `json:"regular"`
	} `json:"de-ru"`
}

type YandexDictEntry struct {
	Text string `json:"text"`
	Pos  struct {
		Code string `json:"code"`
	} `json:"pos"`
	Prdg struct {
		Data []struct {
			Tabs   []string `json:"tabs,omitempty"`
			Tables []struct {
				Tab     int        `json:"tab,omitempty"`
				Headers []string   `json:"headers"`
				Rows    [][]string `json:"rows"`
			} `json:"tables"`
		} `json:"data"`
	} `json:"prdg"`
}

func lookupInYandexDict(word string) (*YandexLookupResult, error) {
	resp, err := http.Get("https://dictionary.yandex.net/dicservice.json/lookupMultiple" +
		"?ui=ru" +
		"&srv=tr-text" +
		"&flags=15783" +
		"&dict=de-ru.regular" +
		"&lang=de-ru" +
		"&text=" + url.QueryEscape(word))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response YandexLookupResult
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	return &response, nil
}

func lookupDictEntry(word, entryCode string) (*YandexDictEntry, error) {
	lookupResult, err := lookupInYandexDict(word)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup entry: %w", err)
	}

	entries := lookupResult.DeRu.Regular
	foundNo := 0
	for i := 0; i < len(entries); i++ {
		if entries[i].Pos.Code != entryCode {
			continue
		}
		entries[foundNo] = entries[i]
		foundNo++
	}
	entries = entries[:foundNo]

	switch foundNo {
	case 0:
		return nil, ErrNoDictEntries
	case 1:
		return &entries[0], nil
	default:
		return nil, fmt.Errorf("%w: got %d, expected 1", ErrMultipleDictEntries, foundNo)
	}
}

func isRegularPresentIndicative(infinitive, pronoun, conjugation string) bool {
	expected := strings.TrimSuffix(infinitive, "n")
	expected = strings.TrimSuffix(expected, "e")
	switch strings.ToLower(pronoun) {
	case "ich":
		expected = expected + "e"
	case "du":
		if strings.HasSuffix(expected, "t") {
			expected = expected + "e"
		}
		expected = expected + "st"
	case "er/sie/es", "ihr":
		if strings.HasSuffix(expected, "t") {
			expected = expected + "e"
		}
		expected = expected + "t"
	case "wir", "sie/sie":
		expected = infinitive
	}

	if expected == conjugation {
		// This is a simple regular verb
		return true
	}
	if strings.Count(conjugation, " ") == 1 {
		// This is a trennbares Verb
		parts := strings.Split(conjugation, " ")
		root := parts[0]
		prefix := parts[1]
		if expected == prefix+root {
			// This is a regular trennbares Verb
			return true
		}
	}

	return false
}

func isRegularPreterite(infinitive string, pronoun string, conjugation string) bool {
	expected := strings.TrimSuffix(infinitive, "n")
	expected = strings.TrimSuffix(expected, "e")
	if len(expected) > 0 {
		switch expected[len(expected)-1] {
		case 'd', 't', 'm', 'n':
			expected = expected + "e"
		default:
			// nop
		}
	}
	switch strings.ToLower(pronoun) {
	case "ich", "er/sie/es":
		expected = expected + "te"
	case "du":
		expected = expected + "test"
	case "ihr":
		expected = expected + "tet"
	case "wir", "sie/sie":
		expected = expected + "ten"
	}

	if expected == conjugation {
		return true
	}
	if strings.Count(conjugation, " ") == 1 {
		// handle trennbare Verben
		parts := strings.Split(conjugation, " ")
		root := parts[0]
		prefix := parts[1]
		if expected == prefix+root {
			return true
		}
	}
	return false
}

func isRegularImperative(_, _, _ string) bool {
	// TODO: implement this function
	return false
}

type VerbConjugationRule struct {
	// NoteField is the field which the conjugation form should be written in the Anki Note
	NoteField string
	// Probability of field being written depending on whether the form is regular or not
	RegularProbability, IrregularProbability float64
	// IsRegular is the function that returns whether a given conjugation form of a verb
	// with the given pronoun is considered regular or not.
	IsRegular func(infinitive, pronounLC, conjugation string) bool
}

// Tab -> Header -> Pronoun -> conjugation rule
var conjugationRules = map[string]map[string]map[string]VerbConjugationRule{
	"настоящее": {
		"Infinitiv": {
			pronounTakeOnlyEntry: {"Infinitiv", 1, 1, func(_, _, _ string) bool {
				return true
			}},
		},
		"Indikativ Präsens": {
			"ich":       {"IndicativPraesensIch", 0.25, 1, isRegularPresentIndicative},
			"du":        {"IndicativPraesensDu", 0.25, 1, isRegularPresentIndicative},
			"er/sie/es": {"IndicativPraesensEr", 0.25, 1, isRegularPresentIndicative},
			"wir":       {"IndicativPraesensWir", 0.05, 1, isRegularPresentIndicative},
			"ihr":       {"IndicativPraesensIhr", 0.15, 1, isRegularPresentIndicative},
			"sie/sie":   {"IndicativPraesensSie", 0.05, 1, isRegularPresentIndicative},
		},
		"Imperativ": {
			"du":  {"ImperativDu", 0.5, 1, isRegularImperative},
			"ihr": {"ImperativIhr", 0.5, 1, isRegularImperative},
		},
	},
	"прошедшее": {
		"Präteritum": {
			"ich":       {"PraeteritumIch", 0.25, 1, isRegularPreterite},
			"du":        {"PraeteritumDu", 0.25, 1, isRegularPreterite},
			"er/sie/es": {"PraeteritumEr", 0.25, 1, isRegularPreterite},
			"wir":       {"PraeteritumWir", 0.05, 1, isRegularPreterite},
			"ihr":       {"PraeteritumIhr", 0.15, 1, isRegularPreterite},
			"sie/sie":   {"PraeteritumSie", 0.05, 1, isRegularPreterite},
		},
	},
}

func parseRawConjugation(raw string) (pronounLC, verbForm string) {
	openIndex, closeIndex := strings.Index(raw, "("), strings.Index(raw, ")")
	if !(0 <= openIndex && openIndex <= closeIndex) {
		return "", ""
	}

	pronounLC = strings.ToLower(raw[openIndex+1 : closeIndex])
	verbForm = strings.TrimSpace(raw[closeIndex+1:])
	return pronounLC, verbForm
}

func conjugateVerb(verbInfinitive string, tags set.Set[string]) ([]ankiCommand, error) {
	var commands []ankiCommand

	entry, err := lookupDictEntry(verbInfinitive, entryCodeVerb)
	if err != nil {
		return nil, err
	}

	switch cnt := len(entry.Prdg.Data); cnt {
	case 0:
		return nil, fmt.Errorf("%w: expected prdg.data array is empty", ErrNoDictEntries)
	case 1:
	// ok -- continue execution
	default:
		return nil, fmt.Errorf("%w: expected prdg.data array to have one entry but got %d", ErrMultipleDictEntries, cnt)
	}
	conjugation := entry.Prdg.Data[0]
	for _, table := range conjugation.Tables {
		tableName := conjugation.Tabs[table.Tab]
		tableRules, ok := conjugationRules[tableName]
		if !ok {
			// There are no conjugation rules for this tab, skip it
			continue
		}

		for columnIdx, rows := range table.Rows {
			columnName := table.Headers[columnIdx]
			columnRules, ok := tableRules[columnName]
			if !ok {
				// There are no conjugation rules for this column, skip it
				continue
			}

			if rule, ok := columnRules[pronounTakeOnlyEntry]; ok {
				if len(rows) != 1 {
					continue
				}
				commands = append(commands, ankiCommand{
					"set_field": map[string]string{rule.NoteField: rows[0]},
				})
				continue
			}

			for _, row := range rows {
				pronounLC, verbForm := parseRawConjugation(row)

				if verbForm == "" {
					// failed to parse the verb form, probably because it's not specified
					continue
				}

				rule, ok := columnRules[pronounLC]
				if !ok {
					// There is no conjugation rule for this pronoun, skip it
					continue
				}

				// Execute the rule

				// In case there are alternative conjugation forms, try to pick the regular one
				if strings.Contains(verbForm, ",") {
					hasRegularForm := false
					for _, variant := range strings.Split(verbForm, ",") {
						variant = strings.TrimSpace(variant)
						if rule.IsRegular(verbInfinitive, pronounLC, variant) {
							verbForm = variant
							hasRegularForm = true
							break
						}
					}
					if !hasRegularForm {
						verbForm = strings.Split(verbForm, ",")[0]
					}
				}

				tag := fmt.Sprintf("conjugation_done:%s", rule.NoteField)
				commands = append(commands, ankiCommand{
					"add_tag": tag,
				})
				if tags.Contains(tag) {
					// the tag for this rule has already been set, do not execute it again
					continue
				}

				rnd := rand.Float64()
				shouldSet := rnd < rule.IrregularProbability
				if rule.IsRegular(verbInfinitive, pronounLC, verbForm) {
					shouldSet = rnd < rule.RegularProbability
				}
				if shouldSet {
					commands = append(commands, ankiCommand{
						"set_field": map[string]string{rule.NoteField: verbForm},
					})
				}
			}
		}
	}
	return commands, nil
}

// Header -> Field name
var nounFormFields = map[string]struct {
	fieldName   string
	probability float64
}{
	// Nominativ is always manually specified in the card, so we should not overwrite it
	"Nominativ": {"SingularNominativ", 0},
	"Genitiv":   {"SingularGenitiv", 0.5},
	"Dativ":     {"SingularDativ", 0.5},
	"Akkusativ": {"SingularAkkusativ", 0.5},
}

var nounFormPattern = regexp.MustCompile(`^\([^/]+/(?P<defArt>[^ ]+)\) (?P<noun>.+)$`)

// processNounForm removes undefined article from the form:
//
// processNounForm("Saft", "(ein/der) Saft") == "der Saft"
func processNounForm(word, form string) string {
	if form == "" || form == "-" {
		return ""
	}

	// TODO: do not panic if the form doesn't match the pattern
	submatch := nounFormPattern.FindStringSubmatch(form)
	article := submatch[nounFormPattern.SubexpIndex("defArt")]
	noun := submatch[nounFormPattern.SubexpIndex("noun")]

	if strings.Contains(noun, ",") {
		// There are multiple possible forms of the word.
		// In this case, we prefer the simple form if it's available, i.e.
		//    (einem/dem) Saft, Safte -> dem Saft
		//    (eines/des) Buchs, Buches -> des Buchs
		options := strings.Split(noun, ", ")
		switch {
		case article == "des" && slices.Contains(options, word+"s"):
			noun = word + "s"
		case article != "des" && slices.Contains(options, word):
			noun = word
		default:
			// nop -- leave all options as is
		}
	}

	return article + " " + noun
}

func addNounForms(noun string, tags set.Set[string]) ([]ankiCommand, error) {
	var commands []ankiCommand

	dictEntry, err := lookupDictEntry(noun, entryCodeNoun)
	if err != nil {
		return nil, fmt.Errorf("failed to look up %q in Yandex.Dict: %w", noun, err)
	}

	entryData := dictEntry.Prdg.Data
	if got := len(entryData); got != 1 {
		return nil, fmt.Errorf("unexpected number of data entries for noun %q: expected 1 but got %d", noun, got)
	}
	entryTables := entryData[0].Tables
	if got := len(entryTables); got != 1 {
		return nil, fmt.Errorf("unexpected number of tables for noun %q: expected 1 but got %d", noun, got)
	}

	formsTable := entryTables[0]
	if want := mapx.Keys(nounFormFields); !slicex.SameElements(formsTable.Headers, want) {
		return nil, fmt.Errorf("unexpected table headers: want %q, got %q", want, formsTable.Headers)
	}
	forms := formsTable.Rows
	if got, want := len(forms), len(formsTable.Headers); want != got {
		return nil, fmt.Errorf("unexpected number of elements in the row: want %d, got %d", want, got)
	}

	for idx, form := range forms {
		if got := len(form); got != /* singular, plural */ 2 {
			return nil, fmt.Errorf("unecpected number of forms in line %d: want 2, got: %d", idx, got)
		}
		singForm := form[0]

		rule := nounFormFields[formsTable.Headers[idx]]
		tag := "noun_form:" + rule.fieldName
		if tags.Contains(tag) {
			// The field has already been set
			continue
		}
		commands = append(commands, ankiCommand{
			"add_tag": tag,
		})

		rnd := rand.Float64()
		if rnd >= rule.probability {
			continue
		}

		processed := processNounForm(dictEntry.Text, singForm)
		if processed == "" {
			// the form is undefined
			continue
		}
		commands = append(commands, ankiCommand{
			"set_field_if_empty": map[string]string{rule.fieldName: processed},
		})
	}
	return commands, nil
}

func findFieldSetting(commands []ankiCommand, field string) (string, bool) {
	var foundValue string
	var valueFound bool
	for _, command := range commands {
		// expected command format: {"COMMAND_KEY": {"FIELD_NAME": "FIELD_VALUE"}}
		for _, commandKey := range []string{"set_field", "set_field_if_empty"} {
			if rawSetCommand, ok := command[commandKey]; ok {
				if setCommand, ok := rawSetCommand.(map[string]string); ok && len(setCommand) == 1 {
					settingField := mapx.Keys(setCommand)[0]
					if settingField == field {
						foundValue = setCommand[settingField]
						valueFound = true
					}
				}
			}
		}
	}
	return foundValue, valueFound
}

func main() {
	if len(os.Args) != 4 {
		log.Fatalf("Unexpected number of CLI argumnets, expected 2 but got: %d", len(os.Args)-1)
	}
	var tagList []string
	if err := json.Unmarshal([]byte(os.Args[3]), &tagList); err != nil {
		log.Fatalf("Failed to parse note tags: %v", err)
	}
	tags := set.FromSlice(tagList...)

	var commands []ankiCommand
	var err error
	switch os.Args[1] {
	case "verb":
		verbInfinitive := os.Args[2]
		commands, err = conjugateVerb(verbInfinitive, tags)
		if errors.Is(err, ErrNoDictEntries) && strings.HasPrefix(verbInfinitive, "sich ") {
			// lookup for reflexive verbs may not work when 'sich' is included in the search query,
			// so we retry with removing 'sich'.
			commands, err = conjugateVerb(strings.TrimPrefix(verbInfinitive, "sich "), tags)
		}
		if foundInfinitive, ok := findFieldSetting(commands, "Infinitiv"); !ok || foundInfinitive != verbInfinitive {
			log.Fatalf(
				"Verb conjugation produced unexpected infinitive setting. got: (%q, %v), expected (%q, true)",
				foundInfinitive, ok, verbInfinitive,
			)
		}
	case "noun":
		commands, err = addNounForms(os.Args[2], tags)
	default:
		log.Fatalf("Unexpected word type: %q", os.Args[1])
	}
	if err != nil {
		log.Fatalf("Execution failed: %+v", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(commands); err != nil {
		log.Fatalf("Failed to write commands to stdout: %+v", err)
	}
}
