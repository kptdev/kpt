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

package gitutil

import (
	"regexp"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
)

// GitExecErrorType is used to enumerate git errors.
type GitExecErrorType int

const (
	// Unknown is used when we can't classify an error into any of the other
	// categories.
	Unknown GitExecErrorType = iota
	// GitExecutableNotFound means the git executable wasn't available.
	GitExecutableNotFound
	// UnknownReference means that provided reference (tag, branch) wasn't
	// found
	UnknownReference
	// HTTPSAuthRequired means we try to access the repo using the https
	// protocol, but the repo required authentication.
	HTTPSAuthRequired
	// RepositoryNotFound means the provided repo uri doesn't seem to point
	// to a valid git repo.
	RepositoryNotFound
	// RepositoryUnavailable means we weren't able to connect to the provided
	// uri.
	RepositoryUnavailable
)

// GitExecError is an error type returned if kpt encounters an error while
// executing a git command. It includes information about the command that
// was executed and the output from git.
type GitExecError struct {
	Type    GitExecErrorType
	Args    []string
	Err     error
	Command string
	Repo    string
	Ref     string
	StdErr  string
	StdOut  string
}

func (e *GitExecError) Error() string {
	b := new(strings.Builder)
	b.WriteString(e.Err.Error())
	b.WriteString(": ")
	b.WriteString(e.StdErr)
	return b.String()
}

// AmendGitExecError provides a way to amend the GitExecError returned by
// the GitLocalRunner.run command.
func AmendGitExecError(err error, f func(e *GitExecError)) {
	var gitExecErr *GitExecError
	if errors.As(err, &gitExecErr) {
		f(gitExecErr)
	}
}

// determineErrorType looks at the output to stderr after executing a git
// command and tries to categorize the error.
func determineErrorType(stdErr string) GitExecErrorType {
	switch {
	case strings.Contains(stdErr, "unknown revision or path not in the working tree"):
		return UnknownReference
	case strings.Contains(stdErr, "could not read Username"):
		return HTTPSAuthRequired
	case strings.Contains(stdErr, "Could not resolve host"):
		return RepositoryUnavailable
	case matches(`fatal: repository '.*' not found`, stdErr):
		return RepositoryNotFound
	}
	return Unknown
}

func matches(pattern, s string) bool {
	matched, err := regexp.Match(pattern, []byte(s))
	if err != nil {
		// This should only return an error if the pattern is invalid, so
		// we just panic if that happens.
		panic(err)
	}
	return matched
}
