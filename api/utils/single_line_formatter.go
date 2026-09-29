// Copyright 2019,2026 The kpt Authors
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

package utils

import (
	"fmt"
	"strings"
)

type SingleLineFormatter struct {
	Title     string   // Label for the output // TODO: this is unnecessary, the call site can just prepend it
	Lines     []string // Lines to be joined
	UseQuote  bool     // Whether to quote each line
	Separator string   // Separator between lines (e.g., comma, space)

	Indent     int // How many spaces to indent the whole text
	LineIndent int // How many (extra) spaces to indent each line
}

func (sf *SingleLineFormatter) String() string {
	mainIndent := strings.Repeat(" ", sf.Indent)

	formatSb := strings.Builder{}

	formatSb.WriteString(mainIndent)
	formatSb.WriteString(strings.Repeat(" ", sf.LineIndent))

	if sf.UseQuote {
		formatSb.WriteString("%q")
	} else {
		formatSb.WriteString("%s")
	}

	var formattedLines []string
	for _, line := range sf.Lines {
		line = strings.ReplaceAll(line, "\n", " ")
		line = strings.TrimSpace(line)
		formattedLines = append(formattedLines, fmt.Sprintf(formatSb.String(), line))
	}

	return fmt.Sprintf("%s%s:\n%s", mainIndent, sf.Title, strings.Join(formattedLines, sf.Separator))
}
