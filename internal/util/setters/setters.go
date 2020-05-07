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
	"path/filepath"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/setters"
	"sigs.k8s.io/kustomize/kyaml/setters2"
	"sigs.k8s.io/kustomize/kyaml/setters2/settersutil"
)

const (
	GcloudProject       = "gcloud.core.project"
	GcloudProjectNumber = "gcloud.project.projectNumber"
)

func PerformSetters(path string) error {

	// auto-fill setters from the environment
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

		rw := &kio.LocalPackageReadWriter{
			PackagePath:           path,
			KeepReaderAnnotations: false,
			IncludeSubpackages:    true,
		}

		setter := &setters.PerformSetters{Name: k, Value: v, SetBy: "kpt"}
		err := kio.Pipeline{
			Inputs:  []kio.Reader{rw},
			Filters: []kio.Filter{setter},
			Outputs: []kio.Writer{rw},
		}.Execute()

		if err != nil {
			return err
		}

		setter2 := &settersutil.FieldSetter{Name: k, Value: v, SetBy: "kpt", ResourcesPath: path, OpenAPIPath: filepath.Join(path, "Kptfile")}
		err = kio.Pipeline{
			Inputs:  []kio.Reader{rw},
			Filters: []kio.Filter{setter2},
			Outputs: []kio.Writer{rw},
		}.Execute()

		if err != nil {
			return err
		}
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

		setter := &setters.PerformSetters{
			Name:  fmt.Sprintf("gcloud.%s", c),
			Value: v,
			SetBy: "kpt",
		}
		rw := &kio.LocalPackageReadWriter{
			PackagePath:           path,
			KeepReaderAnnotations: false,
			IncludeSubpackages:    true,
		}
		err = kio.Pipeline{
			Inputs:  []kio.Reader{rw},
			Filters: []kio.Filter{setter},
			Outputs: []kio.Writer{rw},
		}.Execute()
		if err != nil {
			return err
		}

		setter2 := &settersutil.FieldSetter{
			Name:          fmt.Sprintf("gcloud.%s", c),
			Value:         v,
			SetBy:         "kpt",
			OpenAPIPath:   filepath.Join(path, "Kptfile"),
			ResourcesPath: path,
		}
		err = kio.Pipeline{
			Inputs:  []kio.Reader{rw},
			Filters: []kio.Filter{setter2},
			Outputs: []kio.Writer{rw},
		}.Execute()
		if err != nil {
			return err
		}

		if c == "core.project" && (setter.Count > 0 || setter2.Count > 0) {
			// set the projectNumber if we set the projectID
			projectID = v
		}
	}

	if projectID != "" {
		projectNumber, err := GetProjectNumberFromProjectID(projectID)
		if err != nil {
			return err
		}
		if projectNumber != "" {
			rw := &kio.LocalPackageReadWriter{
				PackagePath:           path,
				KeepReaderAnnotations: false,
				IncludeSubpackages:    true,
			}
			err = kio.Pipeline{
				Inputs: []kio.Reader{rw},
				Filters: []kio.Filter{&setters.PerformSetters{
					Name:  GcloudProjectNumber,
					Value: projectNumber, SetBy: "kpt"}},
				Outputs: []kio.Writer{rw},
			}.Execute()
			if err != nil {
				return err
			}

			err = kio.Pipeline{
				Inputs: []kio.Reader{rw},
				Filters: []kio.Filter{&settersutil.FieldSetter{
					Name:  GcloudProjectNumber,
					Value: projectNumber, SetBy: "kpt", ResourcesPath: path,
					OpenAPIPath: filepath.Join(path, "Kptfile")}},
				Outputs: []kio.Writer{rw},
			}.Execute()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func GetProjectNumberFromProjectID(projectID string) (string, error) {
	gcloudCmd := exec.Command("gcloud",
		"projects", "describe", projectID, "--format", "value(projectNumber)")
	b, err := gcloudCmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get project number for %s, please verify gcloud "+
			"credentials are valid and try again", projectID)
	}
	return strings.TrimSpace(string(b)), nil
}

// DefExists returns true if the setterName exists in Kptfile definitions
func DefExists(resourcePath, setterName string) bool {
	if err := openapi.AddSchemaFromFile(filepath.Join(resourcePath, "Kptfile")); err != nil {
		return false
	}
	ref, err := spec.NewRef(setters2.DefinitionsPrefix + setters2.SetterDefinitionPrefix + setterName)
	if err != nil {
		return false
	}
	setter, _ := openapi.Resolve(&ref)
	return setter != nil
}
