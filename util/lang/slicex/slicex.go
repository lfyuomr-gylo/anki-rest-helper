package slicex

import "maps"

func ElementCounts[S ~[]T, T comparable](s S) map[T]int {
	counts := make(map[T]int)
	for _, e := range s {
		counts[e] = counts[e] + 1
	}
	return counts
}

func SameElements[S ~[]T, T comparable](s1, s2 S) bool {
	return maps.Equal(ElementCounts(s1), ElementCounts(s2))
}
