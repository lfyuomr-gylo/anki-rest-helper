package stringx

func AppendNonEmpty(dst []string, strings ...string) []string {
	for _, s := range strings {
		if s != "" {
			dst = append(dst, s)
		}
	}
	return dst
}

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
