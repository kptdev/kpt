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

package kpt

import (
	"fmt"
	"strings"

	internalpkg "github.com/GoogleContainerTools/kpt/internal/pkg"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TODO: Accept a virtual filesystem or other package abstraction
func UpdateUpstreamFromGit(kptfileContents string, name string, lock kptfilev1.GitLock) (string, error) {

	kptfile, err := internalpkg.DecodeKptfile(strings.NewReader(kptfileContents))
	if err != nil {
		return "", fmt.Errorf("cannot parse Kptfile: %w", err)
	}

	// populate the cloneFrom values so we know where the package came from
	kptfile.UpstreamLock = &kptfilev1.UpstreamLock{
		Type: kptfilev1.GitOrigin,
		Git: &kptfilev1.GitLock{
			Repo:      lock.Repo,
			Directory: lock.Directory,
			Ref:       lock.Ref,
			Commit:    lock.Commit,
		},
	}
	kptfile.Name = name

	b, err := yaml.MarshalWithOptions(kptfile, &yaml.EncoderOptions{SeqIndent: yaml.WideSequenceStyle})
	if err != nil {
		return "", fmt.Errorf("cannot save Kptfile: %w", err)
	}

	return string(b), nil
}
