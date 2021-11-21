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
	CNRMMetricsAnnotation              = "cnrm.cloud.google.com/blueprint"
	DisableKptUsageTrackingEnvVariable = "KPT_DISABLE_USAGE_TRACKING"
)

// Tracker is used to track the usage the of kpt
type Tracker struct {
	// PackagePaths is the package paths to be tracked for usage
	PackagePaths []string

	Resources []*kyaml.RNode

	cmdGroup string
}

// TrackAction invokes Tracker kyaml filter on the resources in input packages paths
func (t *Tracker) TrackAction(cmdGroup string) {
	// users can opt-out by setting the "KPT_DISABLE_USAGE_TRACKING" environment variable
	if os.Getenv(DisableKptUsageTrackingEnvVariable) != "" {
		return
	}

	t.cmdGroup = cmdGroup
	for _, path := range t.PackagePaths {
		inout := &kio.LocalPackageReadWriter{PackagePath: path, PreserveSeqIndent: true, WrapBareSeqNode: true}
		err := kio.Pipeline{
			Inputs:  []kio.Reader{inout},
			Filters: []kio.Filter{kio.FilterAll(t)},
			Outputs: []kio.Writer{inout},
		}.Execute()
		if err != nil {
			// this should be a best effort, do not error if this step fails
			// https://github.com/GoogleContainerTools/kpt/issues/2559
			return
		}
	}

	for _, resource := range t.Resources {
		_, _ = t.Filter(resource)
	}
}

// Filter implements kyaml.Filter
// this filter adds "cnrm.cloud.google.com/blueprint" annotation to the resource
// if the annotation is already present, it appends kpt-<group> suffix
// it uses "default" namespace
func (t *Tracker) Filter(object *kyaml.RNode) (*kyaml.RNode, error) {
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
	if _, err := object.Pipe(kyaml.SetAnnotation(CNRMMetricsAnnotation, recordAction(curAnnoVal, t.cmdGroup))); err != nil {
		return object, nil
	}
	return object, nil
}

// recordAction appends the input group to the annotation to track the usage
// if the group is already present, then it is no-op
func recordAction(curAnnoVal, group string) string {
	if curAnnoVal == "" {
		return fmt.Sprintf("kpt-%s", group)
	}
	if !strings.Contains(curAnnoVal, "kpt-") {
		// just append the value
		return fmt.Sprintf("%s,kpt-%s", curAnnoVal, group)
	}
	// we want to extract the current kpt part from the annotation
	// value and make sure that the input group is added
	// e.g. curAnnoVal: cnrm/landing-zone:networking/v0.4.0,kpt-pkg,blueprints_controller
	curAnnoParts := strings.Split(curAnnoVal, ",")

	// form the new kpt part value
	newKptPart := "kpt"

	for i, curAnnoPart := range curAnnoParts {
		if strings.Contains(curAnnoPart, "kpt") {
			if strings.Contains(curAnnoPart, "pkg") || group == "pkg" {
				newKptPart += "-pkg"
			}
			if strings.Contains(curAnnoPart, "fn") || group == "fn" {
				newKptPart += "-fn"
			}
			if strings.Contains(curAnnoPart, "live") || group == "live" {
				newKptPart += "-live"
			}
			// replace the kpt part with the newly formed part
			curAnnoParts[i] = newKptPart
		}
	}
	return strings.Join(curAnnoParts, ",")
}
