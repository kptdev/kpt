// Copyright 2022 The kpt Authors
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

package kpt

import (
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/util/render"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func TestMemFS(t *testing.T) {
	fs := filesys.MakeFsInMemory()

	if err := fs.MkdirAll("a/b/c/"); err != nil {
		t.Errorf("MkdirAll(\"a/b/c/\") failed %v", err)
	}
	if err := fs.MkdirAll("/d/e/f"); err != nil {
		t.Errorf("MkdirAll(\"/d/e/f\") failed: %v", err)
	}
	if err := fs.WriteFile("/a/b/c/foo.txt", []byte("Hello World")); err != nil {
		t.Errorf("Failed to write file: %v", err)
	}
	if res, err := fs.ReadFile("/a/b/c/foo.txt"); err != nil {
		t.Errorf("Failed to read file: %v", err)
	} else if got, want := string(res), "Hello World"; got != want {
		t.Errorf("unexpected file contents: got %q, want %q", got, want)
	}
}

func TestMemFSRenderBasicPipeline(t *testing.T) {
	resources := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
---
apiVersion: custom.io/v1
kind: Custom
metadata:
  name: custom
spec:
  image: nginx:1.2.3`

	kptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: app
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-namespace:v0.4.1
      configMap:
        namespace: staging
    - image: gcr.io/kpt-fn/set-labels:v0.1.5
      configMap:
        tier: backend`

	expectedResources := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: staging
  labels:
    tier: backend
spec:
  replicas: 3
---
apiVersion: custom.io/v1
kind: Custom
metadata:
  name: custom
  namespace: staging
  labels:
    tier: backend
spec:
  image: nginx:1.2.3
`

	fs := filesys.MakeFsInMemory()
	if err := fs.MkdirAll("a/b/c"); err != nil {
		t.Errorf(`MkdirAll("a/b/c") failed %v`, err)
	}
	if err := fs.WriteFile("/a/b/c/resources.yaml", []byte(resources)); err != nil {
		t.Errorf("Failed to write file: %v", err)
	}
	if err := fs.WriteFile("/a/b/c/Kptfile", []byte(kptfile)); err != nil {
		t.Errorf("Failed to write file: %v", err)
	}
	r := render.Renderer{
		PkgPath:    "/a/b/c",
		FileSystem: fs,
		Runtime:    &runtime{},
	}
	r.RunnerOptions.InitDefaults()
	r.RunnerOptions.ImagePullPolicy = fnruntime.IfNotPresentPull
	_, err := r.Execute(fake.CtxWithDefaultPrinter())
	if err != nil {
		t.Errorf("Failed to render: %v", err)
	}

	if res, err := fs.ReadFile("/a/b/c/resources.yaml"); err != nil {
		t.Errorf("Failed to read file: %v", err)
	} else if got, want := string(res), expectedResources; got != want {
		t.Errorf("unexpected file contents: got %q, want %q", got, want)
	}
}

func TestMemFSRenderSubpkgs(t *testing.T) {
	appResources := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
---
apiVersion: custom.io/v1
kind: Custom
metadata:
  name: custom
spec:
  image: nginx:1.2.3`

	appKptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: app-with-db
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-namespace:v0.4.1
      configMap:
        namespace: staging
    - image: gcr.io/kpt-fn/set-labels:v0.1.5
      configMap:
        tier: db`

	dbResources := `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: db
spec:
  replicas: 3`
	dbKptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: db
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-namespace:v0.4.1
      configMap:
        namespace: db
    - image: gcr.io/kpt-fn/set-labels:v0.1.5
      configMap:
        app: backend`

	expectedAppResources := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: staging
  labels:
    tier: db
spec:
  replicas: 3
---
apiVersion: custom.io/v1
kind: Custom
metadata:
  name: custom
  namespace: staging
  labels:
    tier: db
spec:
  image: nginx:1.2.3
`
	expectedDbResources := `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: db
  namespace: staging
  labels:
    app: backend
    tier: db
spec:
  replicas: 3
`
	fs := filesys.MakeFsInMemory()
	if err := fs.MkdirAll("app/db"); err != nil {
		t.Errorf(`MkdirAll("app/db") failed %v`, err)
	}
	if err := fs.WriteFile("/app/resources.yaml", []byte(appResources)); err != nil {
		t.Errorf("Failed to write file: %v", err)
	}
	if err := fs.WriteFile("/app/Kptfile", []byte(appKptfile)); err != nil {
		t.Errorf("Failed to write file: %v", err)
	}
	if err := fs.WriteFile("/app/db/resources.yaml", []byte(dbResources)); err != nil {
		t.Errorf("Failed to write file: %v", err)
	}
	if err := fs.WriteFile("/app/db/Kptfile", []byte(dbKptfile)); err != nil {
		t.Errorf("Failed to write file: %v", err)
	}

	r := render.Renderer{
		PkgPath:    "/app",
		FileSystem: fs,
		Runtime:    &runtime{},
	}
	r.RunnerOptions.InitDefaults()
	r.RunnerOptions.ImagePullPolicy = fnruntime.IfNotPresentPull

	_, err := r.Execute(fake.CtxWithDefaultPrinter())
	if err != nil {
		t.Errorf("Failed to render: %v", err)
	}

	if res, err := fs.ReadFile("/app/resources.yaml"); err != nil {
		t.Errorf("Failed to read file: %v", err)
	} else if got, want := string(res), expectedAppResources; got != want {
		println(got)
		t.Errorf("unexpected file contents: got %q, want %q", got, want)
	}

	if res, err := fs.ReadFile("/app/db/resources.yaml"); err != nil {
		t.Errorf("Failed to read file: %v", err)
	} else if got, want := string(res), expectedDbResources; got != want {
		t.Errorf("unexpected file contents: got %q, want %q", got, want)
	}
}
