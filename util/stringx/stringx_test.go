package stringx

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAppendNonEmpty(t *testing.T) {
	require.Empty(t, AppendNonEmpty(nil))
	require.Empty(t, AppendNonEmpty(nil, ""))

	require.Equal(t, []string{"foo"}, AppendNonEmpty(nil, "foo"))
	require.Equal(t, []string{"bar", "foo"}, AppendNonEmpty([]string{"bar"}, "foo"))

	// existing empty items should not be removed from the slice, but new ones should not be added
	require.Equal(t, []string{"", "new item"}, AppendNonEmpty([]string{""}, "new item", ""))
}

func TestRemoveEmptyValues(t *testing.T) {
	type smap = map[string]string
	var tests = []struct {
		initial, expected smap
	}{
		{initial: nil, expected: nil},
		{initial: smap{}, expected: smap{}},
		{initial: smap{"": "foo"}, expected: smap{"": "foo"}},
		{initial: smap{"foo": ""}, expected: smap{}},
		{initial: smap{"foo": "bar", "baz": "", "biba": "kuka"}, expected: smap{"foo": "bar", "biba": "kuka"}},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Test_%02d", i), func(t *testing.T) {
			// when:
			RemoveEmptyValues(test.initial)

			// then:
			require.Equal(t, test.expected, test.initial)
		})
	}
}
