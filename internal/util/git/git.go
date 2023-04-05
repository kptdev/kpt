// Copyright 2019 The kpt Authors
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

// Package git contains git repo cloning functions similar to Kustomize's
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
)

// RepoSpec specifies a git repository and a branch and path therein.
type RepoSpec struct {
	// Host, e.g. github.com
	Host string

	// orgRepo name (organization/repoName),
	// e.g. kubernetes-sigs/kustomize
	OrgRepo string

	// Dir where the orgRepo is cloned to.
	Dir string

	// Commit is the commit for the version that was added to Dir.
	Commit string

	// Relative path in the repository, and in the cloneDir,
	// to a Kustomization.
	Path string

	// Branch or tag reference.
	Ref string

	// e.g. .git or empty in case of _git is present
	GitSuffix string
}

// AbsPath is the absolute path to the subdirectory
func (rs RepoSpec) AbsPath() string {
	return filepath.Join(rs.Dir, rs.Path)
}

// CloneSpec returns the string to pass to git to clone
func (rs *RepoSpec) CloneSpec() string {
	if isAzureHost(rs.Host) || isAWSHost(rs.Host) {
		return rs.Host + rs.OrgRepo
	}
	return rs.Host + rs.OrgRepo + rs.GitSuffix
}

// isAzureHost returns true if the repo is an Azure repo
// The format of Azure repo URL is documented
// https://docs.microsoft.com/en-us/azure/devops/repos/git/clone?view=vsts&tabs=visual-studio#clone_url
func isAzureHost(host string) bool {
	return strings.Contains(host, "dev.azure.com") ||
		strings.Contains(host, "visualstudio.com")
}

// isAWSHost returns true if the repo is an AWS repo
// The format of AWS repo URL is documented
// https://docs.aws.amazon.com/codecommit/latest/userguide/regions.html
func isAWSHost(host string) bool {
	return strings.Contains(host, "amazonaws.com")
}

// lookupCommit looks up the sha of the current commit on the repo at the
// provided path.
func LookupCommit(repoPath string) (string, error) {
	const op errors.Op = "git.LookupCommit"
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = repoPath
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return "", errors.E(op, errors.Git, fmt.Errorf("unable to look up commit: %w", err))
	}
	commit := strings.TrimSpace(string(b))
	return commit, nil
}

func (rs *RepoSpec) RepoRef() string {
	repoPath := path.Join(rs.CloneSpec(), rs.Path)
	if rs.Ref != "" {
		return repoPath + fmt.Sprintf("@%s", rs.Ref)
	}
	return repoPath
}
