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
	goerrors "errors"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&gitExecErrorResolver{})
}

const (
	genericGitExecError = `
Error: Failed to execute git command {{ printf "%q " .gitcmd }}
{{- if gt (len .repo) 0 -}}
against repo {{ printf "%q " .repo }}
{{- end }}
{{- if gt (len .ref) 0 -}}
for reference {{ printf "%q " .ref }}
{{- end }}

{{- if or (gt (len .stdout) 0) (gt (len .stderr) 0)}}
{{ printf "\nDetails:" }}
{{- end }}

{{- if gt (len .stdout) 0 }}
{{ printf "%s" .stdout }}
{{- end }}

{{- if gt (len .stderr) 0 }}
{{ printf "%s" .stderr }}
{{- end }}
`

	unknownRefGitExecError = `
Error: Unknown ref {{ printf "%q" .ref }}. Please verify that the reference exists in upstream repo {{ printf "%q" .repo }}.

{{- if or (gt (len .stdout) 0) (gt (len .stderr) 0)}}
{{ printf "\nDetails:" }}
{{- end }}

{{- if gt (len .stdout) 0 }}
{{ printf "%s" .stdout }}
{{- end }}

{{- if gt (len .stderr) 0 }}
{{ printf "%s" .stderr }}
{{- end }}
`
)

// gitExecErrorResolver is an implementation of the ErrorResolver interface
// that can produce error messages for errors of the gitutil.GitExecError type.
type gitExecErrorResolver struct{}

func (*gitExecErrorResolver) Resolve(err error) (string, bool) {
	var gitExecErr *gitutil.GitExecError
	if !goerrors.As(err, &gitExecErr) {
		return "", false
	}
	fullCommand := fmt.Sprintf("git %s %s", gitExecErr.Command,
		strings.Join(gitExecErr.Args, " "))
	tmplArgs := map[string]interface{}{
		"gitcmd": fullCommand,
		"repo":   gitExecErr.Repo,
		"ref":    gitExecErr.Ref,
		"stdout": gitExecErr.StdOut,
		"stderr": gitExecErr.StdErr,
	}
	switch {
	// TODO(mortent): Checking the content of the output at this level seems a bit awkward. We might
	// consider doing this the the gitutil package and use some kind of error code to signal
	// the different error cases to higher levels in the stack.
	case strings.Contains(gitExecErr.StdErr, " unknown revision or path not in the working tree"):
		return ExecuteTemplate(unknownRefGitExecError, tmplArgs)
	default:
		return ExecuteTemplate(genericGitExecError, tmplArgs)
	}
}
