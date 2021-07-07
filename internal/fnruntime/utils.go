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

package fnruntime

import (
	"bytes"
	"io/ioutil"
	"path/filepath"

	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// SaveResults saves results gathered from running the pipeline at specified dir.
func SaveResults(resultsDir string, fnResults *fnresult.ResultList) (string, error) {
	if resultsDir == "" {
		return "", nil
	}
	for _, item := range fnResults.Items {
		item.Image = AddDefaultImagePathPrefix(item.Image)
	}
	filePath := filepath.Join(resultsDir, "results.yaml")
	out := &bytes.Buffer{}

	// use kyaml encoder to ensure consistent indentation
	e := yaml.NewEncoder(out)
	err := e.Encode(fnResults)
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(filePath, out.Bytes(), 0744)
	if err != nil {
		return "", err
	}

	return filePath, nil
}
