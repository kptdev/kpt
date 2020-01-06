// Copyright 2019 Google LLC
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
	"path/filepath"
	"strings"
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
