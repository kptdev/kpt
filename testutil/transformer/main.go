package main

import (
	"bytes"
	"fmt"
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

	// Define the template
	deployment := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Metadata.Name }}
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: generated/{{ .Metadata.Name }}-deployment.yaml
    kpt.dev/kio/mode: 384
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
