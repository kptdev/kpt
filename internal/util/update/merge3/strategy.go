// Copyright 2025 The kpt Authors
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

package merge3

import (
	"github.com/kptdev/kpt/internal/util/attribution"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	mergeSourceAnnotation = "config.kubernetes.io/merge-source"
)

var kyamlAnnos = []string{
	mergeSourceAnnotation,
	kioutil.PathAnnotation,
	kioutil.IndexAnnotation,
	kioutil.LegacyPathAnnotation,  //nolint:staticcheck // SA1019
	kioutil.LegacyIndexAnnotation, //nolint:staticcheck // SA1019
	kioutil.InternalAnnotationsMigrationResourceIDAnnotation,
	attribution.CNRMMetricsAnnotation,
}

// GetHandlingStrategy is an implementation of the ResourceHandler.Handle method from
// kyaml. It is used to decide how a resource should be handled during the
// 3-way merge. This differs from the default implementation in that if a
// resource is deleted from upstream, it will only be deleted from local if
// there is no diff between origin and local.
func GetHandlingStrategy(original, updated, dest *yaml.RNode) filters.ResourceMergeStrategy {
	switch {
	// Keep the resource if added locally.
	case original == nil && updated == nil && dest != nil:
		return filters.KeepDest
	// Add the resource if added in upstream.
	case original == nil && updated != nil && dest == nil:
		return filters.KeepUpdated
	// Do not re-add the resource if deleted from both upstream and local
	case updated == nil && dest == nil:
		return filters.Skip
	// If deleted from upstream, only delete if local fork does not have changes.
	case original != nil && updated == nil:
		if equals(original, dest) {
			return filters.Skip
		} else {
			return filters.KeepDest
		}
	// Do not re-add if deleted from local.
	case original != nil && dest == nil:
		return filters.Skip
	default:
		return filters.Merge
	}
}

func equals(r1, r2 *yaml.RNode) bool {
	// We need to create new copies of the resources since we need to
	// mutate them before comparing them.
	r1Clone, r2Clone := r1.Copy(), r2.Copy()

	// The resources include annotations with information used during the merge
	// process. We need to remove those before comparing the resources.
	stripKyamlAnnos(r1Clone)
	stripKyamlAnnos(r2Clone)

	return r1Clone.MustString() == r2Clone.MustString()
}

func stripKyamlAnnos(n *yaml.RNode) {
	var funcs []yaml.Filter
	for _, a := range kyamlAnnos {
		funcs = append(funcs, yaml.ClearAnnotation(a))
	}
	// ignore error
	_ = n.PipeE(funcs...)
}
