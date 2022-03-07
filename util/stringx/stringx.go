package stringx

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

// RemoveEmptyValues modifies provided map by removing all keys with empty string values.
func RemoveEmptyValues(m map[string]string) {
	keysToRemove := make(map[string]struct{})
	for key, val := range m {
		if val == "" {
			keysToRemove[key] = struct{}{}
		}
	}

	for key := range keysToRemove {
		delete(m, key)
	}
}
