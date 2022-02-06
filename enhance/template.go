package enhance

import (
	"github.com/joomcode/errorx"
	"strings"
)

func substituteVariables(template string, substitutions map[string]string) (string, error) {
	// Determine the longest dollar sequence
	maxDelimiterLength := 0
	curDelimiterLength := 0
	for _, char := range template {
		if char == '$' {
			curDelimiterLength++
		} else {
			if curDelimiterLength > maxDelimiterLength {
				maxDelimiterLength = curDelimiterLength
			}
			curDelimiterLength = 0
		}
	}
	if curDelimiterLength > maxDelimiterLength {
		maxDelimiterLength = curDelimiterLength
	}

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
