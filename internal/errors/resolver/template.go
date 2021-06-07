// Copyright 2021 Google LLC
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

package resolver

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

var baseTemplate = func() *template.Template {
	tmpl := template.New("base")
	tmpl = template.Must(tmpl.Parse(detailsHelperTemplate))
	tmpl = template.Must(tmpl.Parse(nestedErrTemplate))
	return tmpl
}()

var (
	// detailsHelperTemplate is a helper subtemplate that is available to
	// the top-level templates. It is useful when including information from
	// execing other commands in the error message.
	detailsHelperTemplate = `
{{- define "ExecOutputDetails" }}
{{- if or (gt (len .stdout) 0) (gt (len .stderr) 0)}}
{{ printf "\nDetails:" }}
{{- end }}

{{- if gt (len .stdout) 0 }}
{{ printf "%s" .stdout }}
{{- end }}

{{- if gt (len .stderr) 0 }}
{{ printf "%s" .stderr }}
{{- end }}
{{ end }}
`

	// nestedErrTemplate is a helper subtemplate for printing details from
	// a nested error.
	nestedErrTemplate = `
{{- define "NestedErrDetails" }}
{{- if .err  }}
{{- if .err.Err }}
{{- if gt (len .err.Err.Error) 0 }}
{{ printf "\nDetails:" }}
{{ printf "%s" .err.Err.Error }}
{{- end }}
{{- end }}
{{- end }}
{{ end }}
`
)

// ExecuteTemplate takes the provided template string and data, and renders
// the template. If something goes wrong, it panics.
func ExecuteTemplate(text string, data interface{}) string {
	tmpl := template.Must(baseTemplate.Clone())
	template.Must(tmpl.Parse(text))

	var b bytes.Buffer
	execErr := tmpl.Execute(&b, data)
	if execErr != nil {
		panic(fmt.Errorf("error executing template: %w", execErr))
	}
	return strings.TrimSpace(b.String())
}
