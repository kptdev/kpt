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

package addmergecomment

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/util/merge"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// TODO(yuwenma): Those const vars are defined in kpt-functions-sdk/go/fn v0.0.0-20220706221933-7181f451a663+
// we cannot import go/fn directly because the porch/set-namespace uses an older go/fn version. Bumping kpt module alone fails
// kpt CI.
// We should update porch/set-namespace once https://github.com/GoogleContainerTools/kpt-functions-catalog/pull/885 is released.
// and cleanup the const vars below
const (
	upstreamIdentifierFmt = "%s|%s|%s|%s"
	upstreamIdentifier    = "internal.kpt.dev/upstream-identifier"
	unknownNamespace      = "~C"
	defaultNamespace      = "default"
)

// AddMergeComment adds merge comments with format "kpt-merge: namespace/name"
// to all resources in the package
type AddMergeComment struct{}

// Process invokes AddMergeComment kyaml filter on the resources in input packages paths
func Process(paths ...string) error {
	for _, path := range paths {
		inout := &kio.LocalPackageReadWriter{PackagePath: path, PreserveSeqIndent: true, WrapBareSeqNode: true}
		amc := &AddMergeComment{}
		err := kio.Pipeline{
			Inputs:  []kio.Reader{inout},
			Filters: []kio.Filter{kio.FilterAll(amc)},
			Outputs: []kio.Writer{inout},
		}.Execute()
		if err != nil {
			// this should be a best effort, do not error if this step fails
			// https://github.com/GoogleContainerTools/kpt/issues/2559
			return nil
		}
	}
	return nil
}

// addUpstreamAnnotation adds internal.kpt.dev/upstream-identifier annotation to resource.
// In a 3 level package chain (root -> branch -> deployable), the downstream package uses the upstream package meta GKNN
// as its upstream origin, not the upstream its own origin. For example
// root: No `upstreamIdentifier` annotation
// branch: `upstream-identifier=rootGKNN`
// deployable: `upstream-identifier=branchGKNN`
// One known caveat is that upstream meta change can cause downstream origin mismatch. This potentially causes the 3-way merge
// to fail in pkg update step.
func addUpstreamAnnotation(object *kyaml.RNode, mergeComment string) error {
	group, _ := resid.ParseGroupVersion(object.GetApiVersion())
	var name, namespace string
	if strings.Contains(mergeComment, merge.MergeCommentPrefix) {
		nsAndName := merge.NsAndNameForMerge(mergeComment)
		namespace = nsAndName[0]
		name = nsAndName[1]
	} else {
		namespace = object.GetNamespace()
		name = object.GetName()
	}
	// Convert namespace to follow the upstream identifier convention, where
	// - empty string is treated as "default"
	// - unknown custom resource or cluster scoped resource use placeholder "~C"
	if namespace == "" {
		namespace = defaultNamespace
	} else if object.GetNamespace() == resid.TotallyNotANamespace {
		namespace = unknownNamespace
	}
	upstreamIdentifierValue := fmt.Sprintf(upstreamIdentifierFmt, group, object.GetKind(), namespace, name)
	return object.PipeE(kyaml.SetAnnotation(upstreamIdentifier, upstreamIdentifierValue))
}

// Filter implements kyaml.Filter
// this filter adds merge comment with format "kpt-merge: namespace/name" to
// the input resource, if the namespace field doesn't exist on the resource,
// it uses "default" namespace
func (amc *AddMergeComment) Filter(object *kyaml.RNode) (*kyaml.RNode, error) {
	rm, err := object.GetMeta()
	if err != nil {
		// skip adding merge comment if no metadata
		return object, nil
	}
	mf := object.Field(kyaml.MetadataField)
	if object.GetName() == "" && object.GetNamespace() == "" && len(object.GetLabels()) == 0 {
		// skip adding merge comment if empty metadata. Since the intermediate annotations always exist,
		// mf.IsNilOrEmpty cannot tell whether it's empty meta or not.
		// e.g. Empty meta with internal annotations.
		//kind: MyKind
		//spec:
		//  replicas: 3
		//metadata:
		//  annotations:
		//    config.kubernetes.io/index: '0'
		//    config.kubernetes.io/path: 'k8s-cli-982798852.yaml'
		//    internal.config.kubernetes.io/index: '0'
		//    internal.config.kubernetes.io/path: 'k8s-cli-982798852.yaml'
		//    internal.config.kubernetes.io/seqindent: 'compact'
		//    internal.config.kubernetes.io/annotations-migration-resource-id: '0'
		return object, nil
	}

	// Only add merge comment if merge comment does not present
	if !strings.Contains(mf.Key.YNode().LineComment, merge.MergeCommentPrefix) {
		mf.Key.YNode().LineComment = fmt.Sprintf("%s %s/%s", merge.MergeCommentPrefix, rm.Namespace, rm.Name)
	}
	// We will migrate kpt-merge comment to upstream-identifier annotation. As an intermediate stage, this filter
	// preserves the mergeComment behavior to guarantee the backward compatibility.
	if err := addUpstreamAnnotation(object, mf.Key.YNode().LineComment); err != nil {
		return object, nil
	}
	return object, nil
}

// ProcessWithCleanup copies the input directory contents to
// new temp directory and adds merge comment to the resources in directory
// it also returns the cleanup function to clean the created temp directory
func ProcessWithCleanup(path string) (string, func(), error) {
	expected, err := os.MkdirTemp("", "")
	if err != nil {
		return "", nil, err
	}
	err = copyutil.CopyDir(path, expected)
	if err != nil {
		return "", nil, err
	}

	err = Process(expected)
	if err != nil {
		return "", nil, err
	}

	clean := func() {
		os.RemoveAll(expected)
	}

	return expected, clean, nil
}
