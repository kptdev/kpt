package strings

import (
	"fmt"
	"strings"
)

// JoinStringsWithQuotes combines the elements in the string slice into
// a string, with each element inside quotes.
func JoinStringsWithQuotes(strs []string) string {
	b := new(strings.Builder)
	for i, s := range strs {
		b.WriteString(fmt.Sprintf("%q", s))
		if i < (len(strs) - 1) {
			b.WriteString(", ")
		}
	}
	return b.String()
}
