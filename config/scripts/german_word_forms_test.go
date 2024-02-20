package main

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsRegularPresentIndicative(t *testing.T) {
	for idx, tc := range []struct {
		infinitive, pronoun, conjugation string
		isRegular                        bool
	}{
		// simple regular verbs
		{"schicken", "ich", "schicke", true},
		{"schicken", "du", "schickst", true},
		{"schicken", "er/sie/es", "schickt", true},
		{"schicken", "wir", "schicken", true},
		{"schicken", "ihr", "schickt", true},
		{"schicken", "sie/sie", "schicken", true},
		{"warten", "ich", "warte", true},
		{"warten", "du", "wartest", true},
		{"warten", "er/sie/es", "wartet", true},
		{"warten", "wir", "warten", true},
		{"warten", "ihr", "wartet", true},
		{"warten", "sie/sie", "warten", true},

		// simple regular verbs that end with -n
		// NOTE: actual yandex API returns 3 options and 'sammele' is only one of them
		{"sammeln", "ich", "sammele", true},
		{"sammeln", "du", "sammelst", true},
		{"sammeln", "er/sie/es", "sammelt", true},
		{"sammeln", "wir", "sammeln", true},
		{"sammeln", "ihr", "sammelt", true},
		{"sammeln", "sie/sie", "sammeln", true},
		// simple irregular verbs
		{"wollen", "ich", "will", false},
		{"wollen", "du", "willst", false},
		{"wollen", "er/sie/es", "will", false},
		{"sein", "wir", "sind", false},
		{"sein", "ihr", "seid", false},
		{"sein", "sie/sie", "sind", false},
		// trennbar regular verbs
		{"mitbringen", "ich", "bringe mit", true},
		{"mitbringen", "du", "bringst mit", true},
		{"mitbringen", "er/sie/es", "bringt mit", true},
		{"mitbringen", "wir", "bringen mit", true},
		{"mitbringen", "ihr", "bringt mit", true},
		{"mitbringen", "sie/sie", "bringen mit", true},
	} {
		t.Run(fmt.Sprintf("%02d_%s_%s", idx, tc.infinitive, tc.pronoun), func(t *testing.T) {
			require.Equal(t, tc.isRegular, isRegularPresentIndicative(tc.infinitive, tc.pronoun, tc.conjugation))
		})
	}
}

func TestParseRawConjugation(t *testing.T) {
	for idx, tc := range []struct {
		raw, expectedPronounLC, expectedVerbForm string
	}{
		{"(ich) bin", "ich", "bin"},
		{"(du) bist", "du", "bist"},
		{"(er/sie/es) ist", "er/sie/es", "ist"},
		{"(wir) sind", "wir", "sind"},
		{"(ihr) seid", "ihr", "seid"},
		{"(sie/Sie) sind", "sie/sie", "sind"},
	} {
		t.Run(fmt.Sprintf("%02d", idx), func(t *testing.T) {
			pronounLC, verbForm := parseRawConjugation(tc.raw)

			require.Equal(t, tc.expectedPronounLC, pronounLC)
			require.Equal(t, tc.expectedVerbForm, verbForm)
		})
	}
}

func TestProcessNounForm(t *testing.T) {
	for idx, tc := range []struct {
		word, form, want string
	}{
		{"Saft", "(ein/der) Saft", "der Saft"},
		{"Saft", "(eines/des) Saftes, Safts", "des Safts"},
		{"Buch", "(einem/dem) Buch, Buche", "dem Buch"},
	} {
		t.Run(fmt.Sprintf("Case_%02d", idx), func(t *testing.T) {
			if got := processNounForm(tc.word, tc.form); got != tc.want {
				t.Errorf("processNounForm(%q, %q) = %q, want %q", tc.word, tc.form, got, tc.want)
			}
		})
	}
}
