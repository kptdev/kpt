package cmdrender

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"testing/fstest"

	"github.com/mholt/archiver/v4"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

var deployment = `# deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: in-memory-nginx-deployment
spec:
  replicas: 3
`

var kptFile = `# Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: app
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-namespace:v0.1.3
      configMap:
        namespace: staging
`

func inMemPkg() fs.FS {

	rootFS := fstest.MapFS{
		"deployment.yaml": {Data: []byte(deployment)},
		"Kptfile":         {Data: []byte(kptFile)},
	}

	err := fs.WalkDir(rootFS, ".", func(path string, d fs.DirEntry, err error) error {
		fmt.Println(path)
		return nil
	})
	if err != nil {
		panic(err)
	}

	return rootFS
}

func openArchive(filename string) (fs.FS, error) {
	fsys, err := archiver.FileSystem(filename)
	if err != nil {
		return nil, err
	}

	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fmt.Println("Walking:", path, "Dir?", d.IsDir())
		if path == "." {
			// root directory
			return nil
		}
		if filepath.Base(path) == ".git" {
			fmt.Println("Skipping:", path)
			return fs.SkipDir
		}
		return nil
	})
	return fsys, nil
}

// resourceListAsFS abstracts out a resourcelist as fs.FS.
func resourceListAsFS(r io.Reader) (fs.FS, error) {
	br := &kio.ByteReader{
		Reader:            r,
		PreserveSeqIndent: true,
		WrapBareSeqNode:   true,
	}
	xs, err := br.Read()
	if err != nil {
		return nil, err
	}

	rootFS := fstest.MapFS{}

	for _, rs := range xs {
		// rs points to *yaml.RNode
		path, _, err := kioutil.GetFileAnnotations(rs)
		if err != nil {
			return nil, err
		}
		content := rs.MustString()
		rootFS[path] = &fstest.MapFile{Data: []byte(content)}
	}
	return rootFS, nil
}

var rl = bytes.NewBufferString(`apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: app
    annotations:
      config.kubernetes.io/index: '0'
      config.kubernetes.io/path: 'Kptfile'
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'Kptfile'
      internal.config.kubernetes.io/seqindent: 'wide'
  pipeline:
    mutators:
    - image: gcr.io/kpt-fn/set-namespace:v0.1.3
      configMap:
        namespace: staging
    - image: gcr.io/kpt-fn/set-labels:v0.1.4
      configMap:
        tier: backend
- # Copyright 2021 Google LLC
  #
  # Licensed under the Apache License, Version 2.0 (the "License");
  # you may not use this file except in compliance with the License.
  # You may obtain a copy of the License at
  #
  #      http://www.apache.org/licenses/LICENSE-2.0
  #
  # Unless required by applicable law or agreed to in writing, software
  # distributed under the License is distributed on an "AS IS" BASIS,
  # WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  # See the License for the specific language governing permissions and
  # limitations under the License.
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: nginx-deployment
    annotations:
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'deployment.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
  spec:
    replicas: 3
- apiVersion: custom.io/v1
  kind: Custom
  metadata:
    name: custom
    annotations:
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'custom.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
  spec:
    image: nginx:1.2.3
`)
