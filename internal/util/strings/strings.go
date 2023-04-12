// Copyright 2021 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
