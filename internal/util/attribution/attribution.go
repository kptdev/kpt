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

package attribution

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	CNRMMetricsAnnotation            = "cnrm.cloud.google.com/blueprint"
	DisableKptAttributionEnvVariable = "KPT_DISABLE_ATTRIBUTION"
)

// Attributor is used to attribute the kpt action on resources
type Attributor struct {
	// PackagePaths is the package paths to add the attribution annotation
	PackagePaths []string

	// Resources to add the attribution annotation
	Resources []*kyaml.RNode

	// CmdGroup is the command groups in kpt, e.g., pkg, fn, live
	CmdGroup string
}

// Process invokes Attribution kyaml filter on the resources in input packages paths
func (a *Attributor) Process() {
	// users can opt-out by setting the "KPT_DISABLE_ATTRIBUTION" environment variable
	if os.Getenv(DisableKptAttributionEnvVariable) != "" {
		return
	}

	if a.CmdGroup == "" {
		return
	}

	for _, path := range a.PackagePaths {
		inout := &kio.LocalPackageReadWriter{PackagePath: path, PreserveSeqIndent: true, WrapBareSeqNode: true}
		err := kio.Pipeline{
			Inputs:  []kio.Reader{inout},
			Filters: []kio.Filter{kio.FilterAll(a)},
			Outputs: []kio.Writer{inout},
		}.Execute()
		if err != nil {
			// this should be a best effort, do not error if this step fails
			// https://github.com/GoogleContainerTools/kpt/issues/2559
			return
		}
	}

	for _, resource := range a.Resources {
		_, _ = a.Filter(resource)
	}
}

// Filter implements kyaml.Filter
// this filter adds "cnrm.cloud.google.com/blueprint" annotation to the resource
// if the annotation is already present, it appends kpt-<cmdGroup> suffix
// it uses "default" namespace
func (a *Attributor) Filter(object *kyaml.RNode) (*kyaml.RNode, error) {
	// users can opt-out by setting the "KPT_DISABLE_ATTRIBUTION" environment variable
	if os.Getenv(DisableKptAttributionEnvVariable) != "" {
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
	if _, err := object.Pipe(kyaml.SetAnnotation(CNRMMetricsAnnotation, recordAction(curAnnoVal, a.CmdGroup))); err != nil {
		return object, nil
	}
	return object, nil
}

// recordAction appends the input cmdGroup to the annotation to attribute the usage
// if the cmdGroup is already present, then it is no-op
func recordAction(curAnnoVal, cmdGroup string) string {
	if curAnnoVal == "" {
		return fmt.Sprintf("kpt-%s", cmdGroup)
	}
	if !strings.Contains(curAnnoVal, "kpt-") {
		// just append the value
		return fmt.Sprintf("%s,kpt-%s", curAnnoVal, cmdGroup)
	}
	// we want to extract the current kpt part from the annotation
	// value and make sure that the input cmdGroup is added
	// e.g. curAnnoVal: cnrm/landing-zone:networking/v0.4.0,kpt-pkg,blueprints_controller
	curAnnoParts := strings.Split(curAnnoVal, ",")

	// form the new kpt part value
	newKptPart := []string{"kpt"}

	for i, curAnnoPart := range curAnnoParts {
		if strings.Contains(curAnnoPart, "kpt") {
			if strings.Contains(curAnnoPart, "pkg") || cmdGroup == "pkg" {
				newKptPart = append(newKptPart, "pkg")
			}
			if strings.Contains(curAnnoPart, "fn") || cmdGroup == "fn" {
				newKptPart = append(newKptPart, "fn")
			}
			if strings.Contains(curAnnoPart, "live") || cmdGroup == "live" {
				newKptPart = append(newKptPart, "live")
			}
			// replace the kpt part with the newly formed part
			curAnnoParts[i] = strings.Join(newKptPart, "-")
		}
	}
	return strings.Join(curAnnoParts, ",")
}
