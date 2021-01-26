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

package pkgbuilder

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	kptfileutil "github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	deploymentResourceManifest = `
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: myspace
  name: mysql-deployment
spec:
  replicas: 3
  foo: bar
  template:
    spec:
      containers:
      - name: mysql
        image: mysql:1.7.9
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

type resourceInfoWithSetters struct {
	resourceInfo resourceInfo
	setterRefs   []SetterRef
	mutators     []yaml.Filter
}

// Pkg represents a package that can be created on the file system
// by using the Build function
type Pkg struct {
	Name string

	Kptfile *Kptfile

	resources []resourceInfoWithSetters

	files map[string]string

	subPkgs []*Pkg
}

func NewKptfile() *Kptfile {
	return &Kptfile{}
}

// Kptfile represents the Kptfile of a package.
type Kptfile struct {
	Setters []Setter
	Repo    string
	Ref     string
}

// WithUpstream adds information about the upstream information to the Kptfile.
// The upstream section of the Kptfile is only added if this information is
// provided.
func (k *Kptfile) WithUpstream(repo, ref string) *Kptfile {
	k.Repo = repo
	k.Ref = ref
	return k
}

// WithSetters adds information about the setters for a Kptfile.
func (k *Kptfile) WithSetters(setters ...Setter) *Kptfile {
	k.Setters = setters
	return k
}

// Setter contains the properties required for adding a setter to the
// Kptfile.
type Setter struct {
	Name  string
	Value string
	IsSet bool
}

// NewSetter creates a new setter that is not marked as set
func NewSetter(name, value string) Setter {
	return Setter{
		Name:  name,
		Value: value,
	}
}

// NewSetSetter creates a new setter that is marked as set.
func NewSetSetter(name, value string) Setter {
	return Setter{
		Name:  name,
		Value: value,
		IsSet: true,
	}
}

// SetterRef specifies the information for creating a new reference to
// a setter in a resource.
type SetterRef struct {
	Path []string
	Name string
}

// NewSetterRef creates a new setterRef with the given name and path.
func NewSetterRef(name string, path ...string) SetterRef {
	return SetterRef{
		Path: path,
		Name: name,
	}
}

// NewPackage creates a new package for testing.
func NewPackage(name string) *Pkg {
	return &Pkg{
		Name:  name,
		files: make(map[string]string),
	}
}

// WithKptfile configures the current package to have a Kptfile. Only
// zero or one Kptfiles are accepted.
func (p *Pkg) WithKptfile(kf ...*Kptfile) *Pkg {
	if len(kf) > 1 {
		panic("only 0 or 1 Kptfiles are allowed")
	}
	if len(kf) == 0 {
		p.Kptfile = NewKptfile()
	} else {
		p.Kptfile = kf[0]
	}
	return p
}

// WithResource configures the package to include the provided resource
func (p *Pkg) WithResource(resourceName string, mutators ...yaml.Filter) *Pkg {
	resourceInfo, ok := resources[resourceName]
	if !ok {
		panic(fmt.Errorf("unknown resource %s", resourceName))
	}
	p.resources = append(p.resources, resourceInfoWithSetters{
		resourceInfo: resourceInfo,
		setterRefs:   []SetterRef{},
		mutators:     mutators,
	})
	return p
}

// WithResourceAndSetters configures the package to have the provided resource.
// It also allows for specifying setterRefs for the resource and a set of
// mutators that will update the content of the resource.
func (p *Pkg) WithResourceAndSetters(resourceName string, setterRefs []SetterRef, mutators ...yaml.Filter) *Pkg {
	resourceInfo, ok := resources[resourceName]
	if !ok {
		panic(fmt.Errorf("unknown resource %s", resourceName))
	}
	p.resources = append(p.resources, resourceInfoWithSetters{
		resourceInfo: resourceInfo,
		setterRefs:   setterRefs,
		mutators:     mutators,
	})
	return p
}

func (p *Pkg) WithFile(name, content string) *Pkg {
	p.files[name] = content
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

func buildRecursive(path string, pkg *Pkg) error {
	pkgPath := filepath.Join(path, pkg.Name)
	err := os.Mkdir(pkgPath, 0700)
	if err != nil {
		return err
	}

	if pkg.Kptfile != nil {
		content := buildKptfile(pkg)

		err := ioutil.WriteFile(filepath.Join(pkgPath, kptfileutil.KptFileName),
			[]byte(content), 0600)
		if err != nil {
			return err
		}
	}

	for _, ri := range pkg.resources {
		m := ri.resourceInfo.manifest
		r := yaml.MustParse(m)
		for _, setterRef := range ri.setterRefs {
			n, err := r.Pipe(yaml.PathGetter{
				Path: setterRef.Path,
			})
			if err != nil {
				return err
			}
			n.YNode().LineComment = fmt.Sprintf(`{"$openapi":"%s"}`, setterRef.Name)
		}

		for _, m := range ri.mutators {
			if err := r.PipeE(m); err != nil {
				return err
			}
		}

		filePath := filepath.Join(pkgPath, ri.resourceInfo.filename)
		err = ioutil.WriteFile(filePath, []byte(r.MustString()), 0600)
		if err != nil {
			return err
		}
	}

	for name, content := range pkg.files {
		filePath := filepath.Join(pkgPath, name)
		_, err := os.Stat(filePath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("file %s already exists", name)
		}
		err = ioutil.WriteFile(filePath, []byte(content), 0600)
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

var kptfileTemplate = `
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: {{.Name}}
{{- if gt (len .Kptfile.Setters) 0 }}
openAPI:
  definitions:
{{- range .Kptfile.Setters }}
    io.k8s.cli.setters.{{.Name}}:
      x-k8s-cli:
        setter:
          name: {{.Name}}
          value: {{.Value}}
{{- if eq .IsSet true }}
          isSet: true
{{- end }}
{{- end }}
{{- end }}
{{- if gt (len .Kptfile.Repo) 0 }}
upstream:
  type: git
  git:
    ref: {{.Kptfile.Ref}}
    repo: {{.Kptfile.Repo}}
{{- end }}
`

func buildKptfile(pkg *Pkg) string {
	tmpl, err := template.New("test").Parse(kptfileTemplate)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, pkg)
	if err != nil {
		panic(err)
	}
	result := buf.String()
	return result
}

func ExpandPkg(t *testing.T, pkg *Pkg) string {
	if pkg.Name == "" {
		pkg.Name = "base"
	}
	dir, err := ioutil.TempDir("", "test-kpt-builder-")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	err = pkg.Build(dir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return filepath.Join(dir, pkg.Name)
}
