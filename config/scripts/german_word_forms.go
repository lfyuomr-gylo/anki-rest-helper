// Usage:
//   go run ./german_word_forms.go 'word type' 'word'

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	entryCodeNoun = "nn"
	entryCodeVerb = "vrb"
)

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

	if foundNo != 1 {
		return nil, fmt.Errorf("unexpected number of dict entries of type %q: expected 1 but found %d", entryCode, foundNo)
	}
	return &entries[0], nil
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
		"Indikativ Präsens": {
			"ich":       {"IndicativPraesensIch", 0.25, 1, isRegularPresentIndicative},
			"du":        {"IndicativPraesensDu", 0.25, 1, isRegularPresentIndicative},
			"er/sie/es": {"IndicativPraesensEr", 0.25, 1, isRegularPresentIndicative},
			"wir":       {"IndicativPraesensWir", 0.05, 1, isRegularPresentIndicative},
			"ihr":       {"IndicativPraesensIhr", 0.15, 1, isRegularPresentIndicative},
			"sie/sie":   {"IndicativPraesensSie", 0.05, 1, isRegularPresentIndicative},
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

func conjugateVerb(verbInfinitive string) ([]map[string]any, error) {
	var commands []map[string]any

	entry, err := lookupDictEntry(verbInfinitive, entryCodeVerb)
	if err != nil {
		return nil, err
	}

	if cnt := len(entry.Prdg.Data); cnt != 1 {
		return nil, fmt.Errorf("expected prdg.data array to have one entry but got %d", cnt)
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

				rnd := rand.Float64()
				shouldSet := rnd < rule.IrregularProbability
				if rule.IsRegular(verbInfinitive, pronounLC, verbForm) {
					shouldSet = rnd < rule.RegularProbability
				}
				if shouldSet {
					commands = append(commands, map[string]any{
						"set_field": map[string]string{rule.NoteField: verbForm},
					})
				}
				commands = append(commands, map[string]any{
					"add_tag": fmt.Sprintf("conjugation_done:%s", rule.NoteField),
				})
			}
		}
	}
	return commands, nil
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Unexpected number of CLI argumnets, expected 2 but got: %d", len(os.Args)-1)
	}

	var commands any
	var err error
	switch os.Args[1] {
	case "verb":
		commands, err = conjugateVerb(os.Args[2])
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
