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

package kpt

import (
	"fmt"
	"strings"

	internalpkg "github.com/GoogleContainerTools/kpt/internal/pkg"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TODO: Accept a virtual filesystem or other package abstraction
func UpdateUpstream(kptfileContents string, name string, upstream kptfilev1.Upstream, lock kptfilev1.UpstreamLock) (string, error) {

	kptfile, err := internalpkg.DecodeKptfile(strings.NewReader(kptfileContents))
	if err != nil {
		return "", fmt.Errorf("cannot parse Kptfile: %w", err)
	}

	// populate the cloneFrom values so we know where the package came from
	kptfile.UpstreamLock = &lock
	kptfile.Upstream = &upstream
	if name != "" {
		kptfile.Name = name
	}

	b, err := yaml.MarshalWithOptions(kptfile, &yaml.EncoderOptions{SeqIndent: yaml.WideSequenceStyle})
	if err != nil {
		return "", fmt.Errorf("cannot save Kptfile: %w", err)
	}

	return string(b), nil
}

func UpdateName(kptfileContents string, name string) (string, error) {
	kptfile, err := internalpkg.DecodeKptfile(strings.NewReader(kptfileContents))
	if err != nil {
		return "", fmt.Errorf("cannot parse Kptfile: %w", err)
	}

	// update the name of the package
	kptfile.Name = name

	b, err := yaml.MarshalWithOptions(kptfile, &yaml.EncoderOptions{SeqIndent: yaml.WideSequenceStyle})
	if err != nil {
		return "", fmt.Errorf("cannot save Kptfile: %w", err)
	}

	return string(b), nil
}

// TODO: accept a virtual filesystem
func UpdateKptfileUpstream(name string, contents map[string]string, upstream kptfilev1.Upstream, lock kptfilev1.UpstreamLock) error {
	kptfile, found := contents[kptfilev1.KptFileName]
	if !found {
		return fmt.Errorf("package %q is missing Kptfile", name)
	}

	kptfile, err := UpdateUpstream(kptfile, name, upstream, lock)
	if err != nil {
		return fmt.Errorf("failed to update package upstream: %w", err)
	}

	contents[kptfilev1.KptFileName] = kptfile
	return nil
}

func UpdateKptfileName(name string, contents map[string]string) error {
	kptfile, found := contents[kptfilev1.KptFileName]
	if !found {
		return fmt.Errorf("package %q is missing Kptfile", name)
	}

	kptfile, err := UpdateName(kptfile, name)
	if err != nil {
		return fmt.Errorf("failed to update package upstream: %w", err)
	}

	contents[kptfilev1.KptFileName] = kptfile
	return nil
}
