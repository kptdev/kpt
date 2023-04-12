// Copyright 2022 The kpt Authors
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

package git

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// gitAnnotation is the structured data that we store with commits.
// Currently this is stored as a json-encoded blob in the commit message,
// in future we might use git notes or a similar mechanism.
// TODO: Rationalize with OCI data structure?
type gitAnnotation struct {
	// PackagePath is the path of the package we modified.
	// This is useful for disambiguating which package we are modifying in a tree of packages,
	// without having to check file paths.
	PackagePath string `json:"package,omitempty"`

	// WorkspaceName holds the workspaceName of the package revision the commit
	// belongs to.
	WorkspaceName v1alpha1.WorkspaceName `json:"workspaceName,omitempty"`

	// Revision hold the revision of the package revision the commit
	// belongs to.
	Revision string `json:"revision,omitempty"`

	// Task holds the task we performed, if a task caused the commit.
	Task *v1alpha1.Task `json:"task,omitempty"`
}

// ExtractGitAnnotations reads the gitAnnotations from the given commit.
// If no annotation are found, it returns [], nil
// If an invalid annotation is found, it returns an error.
func ExtractGitAnnotations(commit *object.Commit) ([]*gitAnnotation, error) {
	var annotations []*gitAnnotation

	for _, line := range strings.Split(commit.Message, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "kpt:") {
			annotation := &gitAnnotation{}
			b := []byte(strings.TrimPrefix(line, "kpt:"))
			if err := json.Unmarshal(b, annotation); err != nil {
				return nil, fmt.Errorf("failed to unmarshal task command %q: %w", line, err)
			}
			annotations = append(annotations, annotation)
		}
	}

	return annotations, nil
}

// AnnotateCommitMessage adds the gitAnnotation to the commit message.
func AnnotateCommitMessage(message string, annotation *gitAnnotation) (string, error) {
	b, err := json.Marshal(annotation)
	if err != nil {
		return "", fmt.Errorf("error marshaling annotation: %w", err)
	}

	message += "\n\nkpt:" + string(b) + "\n"

	return message, nil
}
