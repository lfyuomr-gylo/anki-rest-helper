package stringx

import "unicode"

// AppendNonEmpty appends all non-empty strings to the specified slice and returns it.
// Empty strings already present in the slice are preserved.
func AppendNonEmpty(dst []string, strings ...string) []string {
	for _, s := range strings {
		if s != "" {
			dst = append(dst, s)
		}
	}
	return dst
}

// RemoveEmptyValuesInPlace modifies provided map by removing all keys with empty string values.
func RemoveEmptyValuesInPlace(m map[string]string) map[string]string {
	keysToRemove := make(map[string]struct{})
	for key, val := range m {
		if val == "" {
			keysToRemove[key] = struct{}{}
		}
	}

	for key := range keysToRemove {
		delete(m, key)
	}

	return m
}

func IsBlank(s string) bool {
	for _, char := range s {
		if !unicode.IsSpace(char) {
			return false
		}
	}
	return true
}
