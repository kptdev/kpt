// Copyright 2020 Google LLC
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

package openapi

import (
	"encoding/json"
	"fmt"
	"github.com/GoogleContainerTools/kpt/internal/util/openapi/augments"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"net/http"
	"runtime"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/openapi/kubernetesapi"
	"sigs.k8s.io/kustomize/kyaml/openapi/kustomizationapi"
)

const (
	BuiltinSchemaVersion = "v1204"
	KubernetesAssetName  = "kubernetesapi/v1204/swagger.json"
	KustomizeAssetName   = "kustomizationapi/swagger.json"

    endpoint = "/openapi"
)

// ConfigureOpenAPI sets the openAPI schema in kyaml.
func ConfigureOpenAPI() error {
	openAPISchema := kubernetesapi.OpenAPIMustAsset[BuiltinSchemaVersion](KubernetesAssetName)
	return ConfigureOpenAPISchema(openAPISchema)
}

func ConfigureOpenAPISchema(openAPISchema []byte) error {
	fmt.Println("configuring openapi schema")
	openapi.SuppressBuiltInSchemaUse()
	openAPISchema, err := addExtensionsToBuiltinTypes(openAPISchema)
	if err != nil {
		return err
	}
	if err := openapi.AddSchema(openAPISchema); err != nil {
		return err
	}
	// Kustomize schema should always be added
	return openapi.AddSchema(kustomizationapi.MustAsset(KustomizeAssetName))
}

// GetJSONSchema returns the JSON OpenAPI schema being used in kyaml
func GetJSONSchema() ([]byte, error) {
	if err := ConfigureOpenAPI(); err != nil {
		return nil, err
	}
	schema := openapi.Schema()
	if schema == nil {
		return nil, nil
	}
	output, err := openapi.Schema().MarshalJSON()
	if err != nil {
		return nil, err
	}
	var jsonSchema map[string]interface{}
	if err := json.Unmarshal(output, &jsonSchema); err != nil {
		return nil, err
	}
	if output, err = json.MarshalIndent(jsonSchema, "", "  "); err != nil {
		return nil, err
	}
	return output, nil
}

func ServerUrl() string {
	var url string
	fmt.Println("os is", runtime.GOOS)
	switch runtime.GOOS {
	case "linux":
		url = "172.17.0.1"
	default:
		url = "host.docker.internal"
	}
	return fmt.Sprintf("http://%s:%s%s", url, "8080", endpoint)
}

func StartLocalServer() error {
	http.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request){
		schema, err := GetJSONSchema()
		if err != nil {
			fmt.Fprintf(w, "error getting schema: %w", err.Error())
		}
		fmt.Printf("endpoint hit: %s\n", endpoint)
		fmt.Fprintf(w, string(schema))
	})

	var err error
	go func () {
		fmt.Println("starting server at port 8080\n")
		err = http.ListenAndServe(":8080", nil) // set listen port
	}()
	return err
}

func addExtensionsToBuiltinTypes(openAPISchema []byte) ([]byte, error) {
	patch, err := jsonpatch.DecodePatch([]byte(augments.JsonPatchBuiltin))
	if err != nil {
		return nil, err
	}
	modified, err := patch.Apply(openAPISchema)
	if err != nil {
		return nil, err
	}
	return modified, nil
}
