// Copyright 2022 Google LLC
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

	"github.com/GoogleContainerTools/kpt/internal/printer/fake"
	"github.com/GoogleContainerTools/kpt/internal/util/render"
	"github.com/google/go-cmp/cmp"
)

const (
	simpleBucketBucket = `
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata: # kpt-merge: config-control/blueprints-project-bucket
  name: blueprints-project-bucket # kpt-set: ${project-id}-${name}
  namespace: config-control # kpt-set: ${namespace}
  annotations:
    cnrm.cloud.google.com/force-destroy: "false"
    cnrm.cloud.google.com/project-id: blueprints-project # kpt-set: ${project-id}
spec:
  storageClass: standard # kpt-set: ${storage-class}
  uniformBucketLevelAccess: true
  versioning:
    enabled: false
`

	simpleBucketKptfile = `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: simple-bucket
  annotations:
    blueprints.cloud.google.com/title: Google Cloud Storage Bucket blueprint
info:
  description: A Google Cloud Storage bucket
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.2.0
      configPath: setters.yaml
`

	simpleBucketSetters = `
apiVersion: v1
kind: ConfigMap
metadata: # kpt-merge: /setters
  name: setters
data:
  name: updated-bucket-name
  namespace: updated-namespace
  project-id: updated-project-id
  storage-class: updated-storage-class
`
)

func TestRenderWithFunctionConfigFile(t *testing.T) {
	t.Skip("kpt renderer does not correctly construct function config")

	fs := &memfs{}
	if err := fs.MkdirAll("/simple-bucket"); err != nil {
		t.Errorf("Failed MkdirAll: %v", err)
	}
	for k, v := range map[string]string{
		"/simple-bucket/bucket.yaml":  simpleBucketBucket,
		"/simple-bucket/Kptfile":      simpleBucketKptfile,
		"/simple-bucket/setters.yaml": simpleBucketSetters,
	} {
		if err := fs.WriteFile(k, []byte(v)); err != nil {
			t.Errorf("Failed creating file %q: %v", k, err)
		}
	}

	r := render.Renderer{
		PkgPath:    "/simple-bucket",
		Runner:     &runner{},
		FileSystem: fs,
	}

	if err := r.Execute(fake.CtxWithDefaultPrinter()); err != nil {
		t.Errorf("Render failed: %v", err)
	}

	got, err := fs.ReadFile("/simple-bucket/bucket.yaml")
	if err != nil {
		t.Errorf("Cannot read \"/simple-bucket/bucket.yaml\": %v", err)
	}

	if diff := cmp.Diff(wantBucketBucket, string(got)); diff != "" {
		t.Errorf("Unexpected result (-want, +got): %s", diff)
	}
}

const (
	inlineBucketBucket = `
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata: # kpt-merge: config-control/blueprints-project-bucket
  name: blueprints-project-bucket # kpt-set: ${project-id}-${name}
  namespace: config-control # kpt-set: ${namespace}
  annotations:
    cnrm.cloud.google.com/force-destroy: "false"
    cnrm.cloud.google.com/project-id: blueprints-project # kpt-set: ${project-id}
spec:
  storageClass: standard # kpt-set: ${storage-class}
  uniformBucketLevelAccess: true
  versioning:
    enabled: false
`

	inlineBucketKptfile = `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: simple-bucket
  annotations:
    blueprints.cloud.google.com/title: Google Cloud Storage Bucket blueprint
info:
  description: A Google Cloud Storage bucket
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.2.0
      configMap:
        name: updated-bucket-name
        namespace: updated-namespace
        project-id: updated-project-id
        storage-class: updated-storage-class
`

	wantBucketBucket = `apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata: # kpt-merge: config-control/blueprints-project-bucket
  name: updated-project-id-updated-bucket-name # kpt-set: ${project-id}-${name}
  namespace: updated-namespace # kpt-set: ${namespace}
  annotations:
    cnrm.cloud.google.com/force-destroy: "false"
    cnrm.cloud.google.com/project-id: updated-project-id # kpt-set: ${project-id}
    cnrm.cloud.google.com/blueprint: 'kpt-fn'
spec:
  storageClass: updated-storage-class # kpt-set: ${storage-class}
  uniformBucketLevelAccess: true
  versioning:
    enabled: false
`
)

func TestRenderWithFunctionConfigInline(t *testing.T) {
	fs := &memfs{}
	if err := fs.MkdirAll("/inline-bucket"); err != nil {
		t.Errorf("Failed MkdirAll: %v", err)
	}
	for k, v := range map[string]string{
		"/inline-bucket/bucket.yaml": inlineBucketBucket,
		"/inline-bucket/Kptfile":     inlineBucketKptfile,
	} {
		if err := fs.WriteFile(k, []byte(v)); err != nil {
			t.Errorf("Failed creating file %q: %v", k, err)
		}
	}

	r := render.Renderer{
		PkgPath:    "/inline-bucket",
		Runner:     &runner{},
		FileSystem: fs,
	}

	if err := r.Execute(fake.CtxWithDefaultPrinter()); err != nil {
		t.Errorf("Render failed: %v", err)
	}

	got, err := fs.ReadFile("/inline-bucket/bucket.yaml")
	if err != nil {
		t.Errorf("Cannot read \"/inline-bucket/bucket.yaml\": %v", err)
	}

	if diff := cmp.Diff(wantBucketBucket, string(got)); diff != "" {
		t.Errorf("Unexpected result (-want, +got): %s", diff)
	}
}
