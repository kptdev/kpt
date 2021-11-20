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

package usage

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	CNRMMetricsAnnotation        = "cnrm.cloud.google.com/blueprint"
	DisableKptMetricsEnvVariable = "DISABLE_KPT_METRICS"
)

// Tracker is used to track the usage the of kpt
type Tracker struct {
	// Group is the command group e.g., pkg, fn, live
	Group string
}

// Process invokes Tracker kyaml filter on the resources in input packages paths
func Process(group string, paths ...string) error {
	for _, path := range paths {
		inout := &kio.LocalPackageReadWriter{PackagePath: path, PreserveSeqIndent: true, WrapBareSeqNode: true}
		ama := &Tracker{Group: group}
		err := kio.Pipeline{
			Inputs:  []kio.Reader{inout},
			Filters: []kio.Filter{kio.FilterAll(ama)},
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
// this filter adds "cnrm.cloud.google.com/blueprint" annotation to the resource
// if the annotation is already present, it appends kpt-<group> suffix
// it uses "default" namespace
func (ama *Tracker) Filter(object *kyaml.RNode) (*kyaml.RNode, error) {
	// users can opt-out by setting the "DISABLE_KPT_METRICS" environment variable
	if os.Getenv(DisableKptMetricsEnvVariable) != "" {
		return object, nil
	}

	// add this annotation to only KCC resource types
	if !strings.Contains(object.GetApiVersion(), ".cnrm.") {
		return object, nil
	}

	curAnnoVal := object.GetAnnotations()[CNRMMetricsAnnotation]
	mf := object.Field(kyaml.MetadataField)
	if mf.IsNilOrEmpty() {
		// skip adding merge comment if empty metadata
		return object, nil
	}
	if _, err := object.Pipe(kyaml.SetAnnotation(CNRMMetricsAnnotation, appendGroup(curAnnoVal, ama.Group))); err != nil {
		return object, nil
	}
	return object, nil
}

// appendGroup appends the input group to the annotation to track the usage
// if the group is already present, then it is no-op
func appendGroup(curAnnoVal, group string) string {
	if curAnnoVal == "" {
		return fmt.Sprintf("kpt-%s", group)
	}
	if !strings.Contains(curAnnoVal, "kpt-") {
		// just append the value
		return fmt.Sprintf("%s,kpt-%s", curAnnoVal, group)
	}
	parts := strings.Split(curAnnoVal, ",")
	val := "kpt"
	for i, part := range parts {
		if strings.Contains(part, "kpt") {
			if strings.Contains(part, "pkg") || group == "pkg" {
				val += "-pkg"
			}
			if strings.Contains(part, "fn") || group == "fn" {
				val += "-fn"
			}
			if strings.Contains(part, "live") || group == "live" {
				val += "-live"
			}
			parts[i] = val
		}
	}
	return strings.Join(parts, ",")
}
