package strings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinStringsWithQuotes(t *testing.T) {
	testCases := map[string]struct {
		slice    []string
		expected string
	}{
		"empty slice": {
			slice:    []string{},
			expected: ``,
		},
		"single element": {
			slice:    []string{"a"},
			expected: `"a"`,
		},
		"two elements": {
			slice:    []string{"a", "b"},
			expected: `"a", "b"`,
		},
		"three elements": {
			slice:    []string{"a", "b", "c"},
			expected: `"a", "b", "c"`,
		},
		"multiple elements": {
			slice:    []string{"a", "b", "c", "d", "e"},
			expected: `"a", "b", "c", "d", "e"`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			res := JoinStringsWithQuotes(tc.slice)
			assert.Equal(t, tc.expected, res)
		})
	}
}
