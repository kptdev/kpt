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

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&gitExecErrorResolver{})
	AddErrorResolver(&fnExecErrorResolver{})
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

{{- template "ExecOutputDetails" . }}
`

	unknownRefGitExecError = `
Error: Unknown ref {{ printf "%q" .ref }}. Please verify that the reference exists in upstream repo {{ printf "%q" .repo }}.

{{- template "ExecOutputDetails" . }}
`

	noGitBinaryError = `
Error: No git executable found. kpt requires git to be installed and available in the path.

{{- template "ExecOutputDetails" . }}
`

	httpsAuthRequired = `
Error: Repository {{ printf "%q" .repo }} requires authentication. kpt does not support this for the 'https' protocol. Please use the 'git' protocol instead.

{{- template "ExecOutputDetails" . }}
`

	repositoryUnavailable = `
Error: Unable to access repository {{ printf "%q" .repo }}.

{{- template "ExecOutputDetails" . }}
`

	repositoryNotFound = `
Error: Repository {{ printf "%q" .repo }} not found.

{{- template "ExecOutputDetails" . }}
`
)

// gitExecErrorResolver is an implementation of the ErrorResolver interface
// that can produce error messages for errors of the gitutil.GitExecError type.
type gitExecErrorResolver struct{}

func (*gitExecErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	var gitExecErr *gitutil.GitExecError
	if !goerrors.As(err, &gitExecErr) {
		return ResolvedResult{}, false
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
	var msg string
	switch gitExecErr.Type {
	case gitutil.UnknownReference:
		msg = ExecuteTemplate(unknownRefGitExecError, tmplArgs)
	case gitutil.GitExecutableNotFound:
		msg = ExecuteTemplate(noGitBinaryError, tmplArgs)
	case gitutil.HTTPSAuthRequired:
		msg = ExecuteTemplate(httpsAuthRequired, tmplArgs)
	case gitutil.RepositoryUnavailable:
		msg = ExecuteTemplate(repositoryUnavailable, tmplArgs)
	case gitutil.RepositoryNotFound:
		msg = ExecuteTemplate(repositoryNotFound, tmplArgs)
	default:
		msg = ExecuteTemplate(genericGitExecError, tmplArgs)
	}
	return ResolvedResult{
		Message:  msg,
		ExitCode: 1,
	}, true
}

// gitExecErrorResolver is an implementation of the ErrorResolver interface
// that can produce error messages for errors of the FnExecError type.
type fnExecErrorResolver struct{}

func (*fnExecErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	kioErr := errors.UnwrapKioError(err)

	var fnErr *errors.FnExecError
	if !goerrors.As(kioErr, &fnErr) {
		return ResolvedResult{}, false
	}
	// TODO: write complete details to a file

	return ResolvedResult{
		Message:  fnErr.String(),
		ExitCode: 1,
	}, true
}
