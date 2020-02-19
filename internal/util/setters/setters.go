// Copyright 2019 Google LLC
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

package setters

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/setters"
)

func PerformSetters(path string) error {
	rw := &kio.LocalPackageReadWriter{
		PackagePath:           path,
		KeepReaderAnnotations: false,
		IncludeSubpackages:    true,
	}

	// auto-fill setters from the environment
	var fltrs []kio.Filter
	for i := range os.Environ() {
		e := os.Environ()[i]
		if !strings.HasPrefix(e, "KPT_SET_") {
			continue
		}
		parts := strings.SplitN(e, "=", 2)
		if len(parts) < 2 {
			continue
		}
		k, v := strings.TrimPrefix(parts[0], "KPT_SET_"), parts[1]
		k = strings.ToLower(k)
		fltrs = append(fltrs, &setters.PerformSetters{Name: k, Value: v, SetBy: "kpt"})
	}

	// auto-fill setters from gcloud
	gcloudConfig := []string{"compute.region", "compute.zone", "core.project"}
	projectID := ""
	for _, c := range gcloudConfig {
		gcloudCmd := exec.Command("gcloud",
			"config", "list", "--format", fmt.Sprintf("value(%s)", c))
		b, err := gcloudCmd.Output()
		if err != nil {
			// don't fail if gcloud fails -- it may not be installed or have this config property
			continue
		}
		v := strings.TrimSpace(string(b))
		if v == "" {
			// don't replace values that aren't set - stick with the defaults as defined in the manifest
			continue
		}
		if c == "core.project" {
			projectID = v
		}
		fltrs = append(fltrs, &setters.PerformSetters{Name: fmt.Sprintf("gcloud.%s", c), Value: v, SetBy: "kpt"})
	}

	if projectID != "" {
		projectNumber, err := GetProjectNumberFromProjectID(projectID)
		if err != nil {
			return err
		}
		if projectNumber != "" {
			fltrs = append(fltrs, &setters.PerformSetters{Name: "gcloud.project.projectNumber", Value: projectNumber, SetBy: "kpt"})
		}
	}

	if len(fltrs) == 0 {
		return nil
	}

	return kio.Pipeline{Inputs: []kio.Reader{rw}, Filters: fltrs, Outputs: []kio.Writer{rw}}.
		Execute()
}

func GetProjectNumberFromProjectID(projectID string) (string, error) {
	gcloudCmd := exec.Command("gcloud",
		"projects", "describe", projectID, "--format", "value(projectNumber)")
	b, err := gcloudCmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "failed to get project number")
	}
	return strings.TrimSpace(string(b)), nil
}
