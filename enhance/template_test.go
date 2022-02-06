package enhance

import (
	"fmt"
	"github.com/joomcode/errorx"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSubstituteVariables_MalformedTemplates(t *testing.T) {
	vars := map[string]string{
		"FOO": "biba",
		"BAR": "kuka",
	}
	invalidTemplates := []string{
		"single dollar sign $ in a string is ambiguous",
		"reference $FOO$ with less dollars than specified somewhere $$ else",
		"undefined $$VARIABLE$$ reference is invalid",
	}

	for i, template := range invalidTemplates {
		t.Run(fmt.Sprintf("Template_%02d", i), func(t *testing.T) {
			// when:
			_, err := substituteVariables(template, vars)

			// then:
			require.Error(t, err)
			require.True(t, errorx.IsOfType(err, errorx.IllegalFormat), "expected malformed_format but got %+v", err)
		})
	}
}

func TestSubstituteVariables_ValidTemplates(t *testing.T) {
	type vars = map[string]string
	testCases := []struct {
		tmpl   string
		vars   vars
		expect string
	}{
		{tmpl: "empty template", vars: nil, expect: "empty template"},
		{tmpl: "$$EMPTY$$template with $ sign", vars: nil, expect: "template with $ sign"}, // EMPTY should be automatically added
		{tmpl: "overridden $$EMPTY$$ value", vars: vars{"EMPTY": "empty"}, expect: "overridden empty value"},
		{tmpl: "$FOO$ and $bar$", vars: vars{"FOO": "pupa", "bar": "lupa"}, expect: "pupa and lupa"},
		{tmpl: "$$FOO$$ $bar$", vars: vars{"FOO": "my foo"}, expect: "my foo $bar$"},
		{tmpl: "reference $$IN$$ the $$MIDDLE$$ of template", vars: vars{"IN": "in", "MIDDLE": "mid"}, expect: "reference in the mid of template"},
	}

	for i, test := range testCases {
		t.Run(fmt.Sprintf("Test_%d", i), func(t *testing.T) {
			// when:
			result, err := substituteVariables(test.tmpl, test.vars)

			// then:
			require.NoError(t, err)
			require.Equal(t, test.expect, result)
		})
	}
}
