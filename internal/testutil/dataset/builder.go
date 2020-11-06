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

package dataset

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	deploymentResourceManifest = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`

	configMapResourceManifest = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: configmap
data:
  foo: bar
`
)

var (
	DeploymentResource = "deployment"
	ConfigMapResource  = "configmap"
	resources          = map[string]resourceInfo{
		DeploymentResource: {
			filename: "deployment.yaml",
			manifest: deploymentResourceManifest,
		},
		ConfigMapResource: {
			filename: "configmap.yaml",
			manifest: configMapResourceManifest,
		},
	}
)

type resourceInfo struct {
	filename string
	manifest string
}

// Pkg represents a package that can be created on the file system
// by using the Build function
type Pkg struct {
	name string

	kptfile bool

	resources []string

	subPkgs []*Pkg
}

// NewPackage creates a new package for testing.
func NewPackage(name string) *Pkg {
	return &Pkg{
		name: name,
	}
}

// WithKptfile configures the current package to have a Kptfile
func (p *Pkg) WithKptfile() *Pkg {
	p.kptfile = true
	return p
}

// WithResource configures the package to include the provided resource
func (p *Pkg) WithResource(resource string) *Pkg {
	p.resources = append(p.resources, resource)
	return p
}

// WithSubPackages adds the provided packages as subpackages to the current
// package
func (p *Pkg) WithSubPackages(ps ...*Pkg) *Pkg {
	p.subPkgs = append(p.subPkgs, ps...)
	return p
}

// Build outputs the current data structure as a set of (nested) package
// in the provided path.
func (p *Pkg) Build(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	return buildRecursive(path, p)
}

// Name returns the name of the current package
func (p *Pkg) Name() string {
	return p.name
}

func buildRecursive(path string, pkg *Pkg) error {
	pkgPath := filepath.Join(path, pkg.name)
	err := os.Mkdir(pkgPath, 0700)
	if err != nil {
		return err
	}

	if pkg.kptfile {
		err := ioutil.WriteFile(filepath.Join(pkgPath, "Kptfile"), []byte(`
apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: kpt
`), 0600)
		if err != nil {
			return err
		}
	}

	for i := range pkg.resources {
		r := pkg.resources[i]
		info, ok := resources[r]
		if !ok {
			return fmt.Errorf("unknown resource %s", r)
		}
		err = ioutil.WriteFile(filepath.Join(pkgPath, info.filename), []byte(info.manifest), 0600)
		if err != nil {
			return err
		}
	}

	for i := range pkg.subPkgs {
		subPkg := pkg.subPkgs[i]
		err = buildRecursive(pkgPath, subPkg)
		if err != nil {
			return err
		}
	}

	return nil
}
