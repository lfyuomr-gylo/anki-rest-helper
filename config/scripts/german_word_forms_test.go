package main

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseRawConjugation(t *testing.T) {
	for idx, tc := range []struct {
		raw, expectedPronounLC, expectedVerbForm string
	}{
		{"(ich) bin", "(ich)", "bin"},
		{"(du) bist", "(du)", "bist"},
		{"(er/sie/es) ist", "(er/sie/es)", "ist"},
		{"(wir) sind", "(wir)", "sind"},
		{"(ihr) seid", "(ihr)", "seid"},
		{"(sie/Sie) sind", "(sie/sie)", "sind"},
	} {
		t.Run(fmt.Sprintf("%02d", idx), func(t *testing.T) {
			pronounLC, verbForm := parseRawConjugation(tc.raw)

			require.Equal(t, tc.expectedPronounLC, pronounLC)
			require.Equal(t, tc.expectedVerbForm, verbForm)
		})
	}
}
