package ankihelper

import (
	"github.com/joomcode/errorx"
	"strings"
)

// substituteVariables implements a dummy template substitution -- it substitutes variable references like $VAR_NAME$.
// If there are dollar signs in the template that should not be confused with variable reference, use more
// consecutive dollar signs, e.g.
//	substituteVariables("$$FOO$$ $FOO$", map[string]string{"FOO": "bar"}) // returns "bar $FOO$".
//
// NOTE: variable reference delimiter is determined as the longest sequence of consecutive dollar signs in the string.
// Thus, templates with a dollar sign adjacent to a variable reference are not supported.
//
// If template contains a dollar sign, it's expected to contain at least one variable reference. To avoid confusion,
// consider using implicit $$EMPTY$$ variable to make sure it doesn't break.
func substituteVariables(template string, substitutions map[string]string) (string, error) {
	// Determine variable reference delimiter
	maxDelimiterLength := longestCharSubsequenceLength(template, '$')
	if maxDelimiterLength == 0 {
		// template doesn't reference any variables, nothing to substitute
		return template, nil
	}
	delimiter := strings.Repeat("$", maxDelimiterLength)

	// Substitute variables
	var result strings.Builder
	for {
		openingDelIdx := strings.Index(template, delimiter)
		if openingDelIdx == -1 {
			break
		}
		varNameIdx := openingDelIdx + len(delimiter)
		varNameLen := strings.Index(template[varNameIdx:], delimiter)
		if varNameLen == -1 {
			return "", errorx.IllegalFormat.New("No closing %q delimiter", delimiter)
		}
		closingDelIdx := varNameIdx + varNameLen

		varName := template[varNameIdx:closingDelIdx]
		varVal, ok := substitutions[varName]
		if !ok {
			switch varName {
			case "EMPTY":
				// this is a special implicitly declared variable to be used in templates without variable references
				// that contain dollar signs.
				varVal, ok = "", true
			default:
				return "", errorx.IllegalFormat.New("Variable %q is not defined", varName)
			}
		}

		// write text preceding the variable reference
		result.WriteString(template[:openingDelIdx])
		// substitute variable
		result.WriteString(varVal)
		// shift template
		template = template[closingDelIdx+len(delimiter):]
	}
	result.WriteString(template) // write remaining template text without variables
	return result.String(), nil
}

func longestCharSubsequenceLength(str string, char rune) int {
	maxSeqLength := 0
	curSeqLength := 0
	for _, curChar := range str {
		if curChar == char {
			curSeqLength++
		} else {
			if curSeqLength > maxSeqLength {
				maxSeqLength = curSeqLength
			}
			curSeqLength = 0
		}
	}
	if curSeqLength > maxSeqLength {
		maxSeqLength = curSeqLength
	}
	return maxSeqLength
}
