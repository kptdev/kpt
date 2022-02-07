// Copyright 2021 Google LLC
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
	"io/ioutil"
	"os"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/merge"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// AddMergeComment adds merge comments with format "kpt-merge: namespace/name"
// to all resources in the package
type AddMergeComment struct{}

// ProcessObsolete invokes AddMergeComment kyaml filter on the resources in input packages paths
func ProcessObsolete(paths ...string) error {
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

// ProcessObsolete invokes AddMergeComment kyaml filter on the resources in input packages paths
func Process(paths ...types.FileSystemPath) error {
	for _, path := range paths {
		inout := &kio.LocalPackageReadWriter{PackagePath: path.Path, PreserveSeqIndent: true, WrapBareSeqNode: true}
		inout.FileSystem.Set(path.FileSystem)
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
	if mf.IsNilOrEmpty() {
		// skip adding merge comment if empty metadata
		return object, nil
	}
	if strings.Contains(mf.Key.YNode().LineComment, merge.MergeCommentPrefix) {
		// skip adding merge comment if merge comment is already present
		return object, nil
	}
	mf.Key.YNode().LineComment = fmt.Sprintf("%s %s/%s", merge.MergeCommentPrefix, rm.Namespace, rm.Name)
	return object, nil
}

// ProcessWithCleanup copies the input directory contents to
// new temp directory and adds merge comment to the resources in directory
// it also returns the cleanup function to clean the created temp directory
func ProcessWithCleanup(path string) (string, func(), error) {
	expected, err := ioutil.TempDir("", "")
	if err != nil {
		return "", nil, err
	}
	err = copyutil.CopyDir(path, expected)
	if err != nil {
		return "", nil, err
	}

	err = ProcessObsolete(expected)
	if err != nil {
		return "", nil, err
	}

	clean := func() {
		os.RemoveAll(expected)
	}

	return expected, clean, nil
}
