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

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"text/template"

	"gopkg.in/yaml.v3"
)

type API struct {
	// Metdata contains the Deployment metadata
	Metadata Metadata `yaml:"metadata""`

	// Replicas is the number of Deployment replicas
	// Defaults to the REPLICAS env var, or 1
	Replicas *int `yaml:"replicas""`

	// Image is the container image
	Image string `yaml:"image"`
}

type Metadata struct {
	// Name is the Deployment Resource and Container name
	Name string `yaml:"name""`
}

func main() {
	// Parse the configuration and decode it into an object
	d := yaml.NewDecoder(bytes.NewBufferString(os.Getenv("API_CONFIG")))
	d.KnownFields(false)
	api := &API{}
	if err := d.Decode(api); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// copy the input resources so we merge our changes
	if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
		panic(err)
	}
	fmt.Println("\n---")

	// Default the Replicas field
	r := os.Getenv("REPLICAS")
	if r != "" && api.Replicas == nil {
		replicas, err := strconv.Atoi(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		api.Replicas = &replicas
	}
	if api.Replicas == nil {
		r := 1
		api.Replicas = &r
	}

	// Define the template.
	// Disable the duck-commands for this generated Resource so that users don't override
	// the generated values.
	deployment := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Metadata.Name }}
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: generated/{{ .Metadata.Name }}-deployment.yaml
    kpt.dev/kio/mode: 384
    kpt.dev/duck/set-image: disabled
    kpt.dev/duck/get-image: disabled
    kpt.dev/duck/set-replicas: disabled
    kpt.dev/duck/get-replicas: disabled
spec:
  replicas: {{ .Replicas }}
  template:
    spec:
      containers:
      - name: {{ .Metadata.Name }}
        image: {{ .Image }}
`

	// Execute the template
	t := template.Must(template.New("deployment").Parse(deployment))
	if err := t.Execute(os.Stdout, api); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
