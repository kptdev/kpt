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

package helloworld

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/functions/examples/util"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	Kind       = "HelloWorld"
	APIVersion = "examples.kpt.dev/v1alpha1"
)

var _ kio.Filter = &HelloWorldFunction{}

// Filter returns a new HelloWorldFunction
func Filter() kio.Filter {
	return &HelloWorldFunction{}
}

// HelloWorldFunction implements the HelloWorld Function
type HelloWorldFunction struct {
	// Kind is the API name.  Must be HelloWorld.
	Kind string `yaml:"kind"`

	// APIVersion is the API version.  Must be examples.kpt.dev/v1alpha1
	APIVersion string `yaml:"apiVersion"`

	// Metadata defines instance metadata.
	Metadata Metadata `yaml:"metadata"`

	// Spec defines the desired declarative configuration.
	Spec Spec `yaml:"spec"`
}

type Metadata struct {
	// Name is the name of the HelloWorld Resources
	Name string `yaml:"name"`

	// Namespace is the namespace of the HelloWorld Resources
	Namespace string `yaml:"namespace"`

	// Labels are labels applied to the HelloWorld Resources
	Labels map[string]string `yaml:"labels"`

	// Annotations are annotations applied to the HelloWorld Resources
	Annotations map[string]string `yaml:"annotations"`
}

type Spec struct {
	Version string `yaml:"version"`

	Port *int32 `yaml:"port"`

	Replicas *int32 `yaml:"replicas"`

	Selector map[string]string `yaml:"selector"`
}

func (f *HelloWorldFunction) init() error {
	if f.Metadata.Name == "" {
		return fmt.Errorf("must specify HelloWorld name")
	}
	if f.Spec.Version == "" {
		f.Spec.Version = "0.1.0"
	}
	if f.Spec.Port == nil {
		var p int32 = 80
		f.Spec.Port = &p
	}
	if *f.Spec.Port <= 0 {
		return fmt.Errorf("HelloWorld spec.port must be greater than 0")
	}

	if f.Spec.Replicas == nil {
		var r int32 = 1
		f.Spec.Replicas = &r
	}
	if *f.Spec.Replicas < 0 {
		return fmt.Errorf("HelloWorld spec.replicas must be greater than or equal to 0")
	}

	if len(f.Spec.Selector) == 0 {
		return fmt.Errorf("HelloWorld spec.selector must be specified")
	}

	if f.Metadata.Labels == nil {
		f.Metadata.Labels = map[string]string{}
	}

	for k, v := range f.Spec.Selector {
		f.Metadata.Labels[k] = v
	}
	return nil
}

func (f *HelloWorldFunction) Filter(inputs []*yaml.RNode) ([]*yaml.RNode, error) {
	if err := f.init(); err != nil {
		return nil, err
	}
	// override input values
	r := util.MustParseAll(
		util.Template{Input: f, Name: "helloworld-deployment", Template: helloWorldDeployment},
		util.Template{Input: f, Name: "helloworld-service", Template: helloWorldService},
	)
	return filters.MergeFilter{}.Filter(append(inputs, r...))
}

const helloWorldDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Metadata.Name}}
  {{- if .Metadata.Namespace}}
  namespace: {{.Metadata.Namespace}}
  {{- end}}
  labels:
  {{- range $k, $v := .Metadata.Labels}}
    {{ $k }}: "{{ $v }}" 
  {{- end}}
spec:
  replicas: {{ .Spec.Replicas }}
  selector:
    matchLabels:
      {{- range $k, $v := .Spec.Selector}}
      {{ $k }}: "{{ $v }}" 
      {{- end}}
  template:
    metadata:
      labels:
        {{- range $k, $v := .Metadata.Labels}}
        {{ $k }}: "{{ $v }}" 
        {{- end}}
    spec:
      containers:
      - name: helloworld-gke
        image: gcr.io/kpt-dev/helloworld-gke:{{.Spec.Version}}
        ports:
        - name: http
          containerPort: {{.Spec.Port}}
        env:
        - name: PORT
          value: "{{.Spec.Port}}"
`

const helloWorldService = `
apiVersion: v1
kind: Service
metadata:
  name: {{.Metadata.Name}}
  {{- if .Metadata.Namespace}}
  namespace: {{.Metadata.Namespace}}
  {{-  end}}
  labels:
  {{- range $k, $v := .Metadata.Labels}}
    {{ $k }}: "{{ $v }}" 
  {{- end}}
spec:
  type: LoadBalancer
  selector:
    {{- range $k, $v := .Spec.Selector}}
    {{ $k }}: "{{ $v }}" 
    {{- end}}
  ports:
  - protocol: TCP
    port: {{.Spec.Port}}
    targetPort: http
`
