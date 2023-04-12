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

	var msg string
	switch gitExecErr.Type {
	case gitutil.UnknownReference:
		msg = fmt.Sprintf("Error: Unknown ref %q. Please verify that the reference exists in upstream repo %q.", gitExecErr.Ref, gitExecErr.Repo)
		msg = msg + "\n" + BuildOutputDetails(gitExecErr.StdOut, gitExecErr.StdErr)

	case gitutil.GitExecutableNotFound:
		msg = "Error: No git executable found. kpt requires git to be installed and available in the path."
		msg = msg + "\n" + BuildOutputDetails(gitExecErr.StdOut, gitExecErr.StdErr)

	case gitutil.HTTPSAuthRequired:
		msg = fmt.Sprintf("Error: Repository %q requires authentication.", gitExecErr.Repo)
		msg += " kpt does not support this for the 'https' protocol."
		msg += " Please use the 'git' protocol instead."
		msg = msg + "\n" + BuildOutputDetails(gitExecErr.StdOut, gitExecErr.StdErr)

	case gitutil.RepositoryUnavailable:
		msg = fmt.Sprintf("Error: Unable to access repository %q.", gitExecErr.Repo)
		msg = msg + "\n" + BuildOutputDetails(gitExecErr.StdOut, gitExecErr.StdErr)

	case gitutil.RepositoryNotFound:
		msg = fmt.Sprintf("Error: Repository %q not found.", gitExecErr.Repo)
		msg = msg + "\n" + BuildOutputDetails(gitExecErr.StdOut, gitExecErr.StdErr)
	default:
		msg = fmt.Sprintf("Error: Failed to execute git command %q", fullCommand)
		if gitExecErr.Repo != "" {
			msg += fmt.Sprintf(" against repo %q", gitExecErr.Repo)
		}
		if gitExecErr.Ref != "" {
			msg += fmt.Sprintf(" for reference %q", gitExecErr.Ref)
		}
		msg = msg + "\n" + BuildOutputDetails(gitExecErr.StdOut, gitExecErr.StdErr)
	}
	return ResolvedResult{
		Message: msg,
	}, true
}

func BuildOutputDetails(stdout string, stderr string) string {
	var sb strings.Builder
	if len(stdout) > 0 || len(stderr) > 0 {
		sb.WriteString("\nDetails:\n")
	}
	if len(stdout) > 0 {
		sb.WriteString(stdout)
	}
	if len(stderr) > 0 {
		sb.WriteString(stderr)
	}
	return sb.String()
}
