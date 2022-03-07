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

package engine

import (
	"context"
	"fmt"
	"path"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type initPackageMutation struct {
	name string
	spec api.PackageInitTaskSpec
}

var _ mutation = &initPackageMutation{}

func (m *initPackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	kptfile := kptfilev1.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1.KptFileAPIVersion,
				Kind:       kptfilev1.KptFileKind,
			},
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: m.name,
				},
			},
		},
		Info: &kptfilev1.PackageInfo{
			Site:        m.spec.Site,
			Description: m.spec.Description,
			Keywords:    m.spec.Keywords,
		},
	}

	b, err := yaml.MarshalWithOptions(kptfile, &yaml.EncoderOptions{SeqIndent: yaml.WideSequenceStyle})
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to serialize Kptfile: %w", err)
	}

	if resources.Contents == nil {
		resources.Contents = map[string]string{}
	}

	kptfilePath := path.Join(m.spec.Subpackage, kptfilev1.KptFileName)
	if _, found := resources.Contents[kptfilePath]; found {
		return repository.PackageResources{}, nil, fmt.Errorf("package %q already initialized", m.name)
	}

	resources.Contents[kptfilePath] = string(b)

	return resources, &api.Task{}, nil
}
