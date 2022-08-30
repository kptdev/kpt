// Copyright 2022 Google LLC
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

	"github.com/GoogleContainerTools/kpt/porch/pkg/meta"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ExtractGitAnnotations reads the gitAnnotations from the given commit.
// If no annotation are found, it returns [], nil
// If an invalid annotation is found, it returns an error.
func ExtractGitAnnotations(commit *object.Commit) ([]*meta.Annotation, error) {
	var annotations []*meta.Annotation

	for _, line := range strings.Split(commit.Message, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "kpt:") {
			b := []byte(strings.TrimPrefix(line, "kpt:"))
			annotation, err := meta.ParseAnnotation(b)
			if err != nil {
				return nil, err
			}
			annotations = append(annotations, annotation)
		}
	}

	return annotations, nil
}

// AnnotateCommitMessage adds the gitAnnotation to the commit message.
func AnnotateCommitMessage(message string, annotation *meta.Annotation) (string, error) {
	b, err := json.Marshal(annotation)
	if err != nil {
		return "", fmt.Errorf("error marshaling annotation: %w", err)
	}

	message += "\n\nkpt:" + string(b) + "\n"

	return message, nil
}
