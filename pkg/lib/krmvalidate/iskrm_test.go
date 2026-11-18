// Copyright 2026 The kpt Authors
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

package krmvalidate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func mustParse(t *testing.T, s string) *yaml.RNode {
	t.Helper()
	n, err := yaml.Parse(s)
	assert.NoError(t, err)
	return n
}

func TestIsKRM_ValidResource(t *testing.T) {
	n := mustParse(t, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-cm
`)
	assert.NoError(t, IsKRM(n))
}

func TestIsKRM_MissingAPIVersion(t *testing.T) {
	n := mustParse(t, `
kind: ConfigMap
metadata:
  name: my-cm
`)
	err := IsKRM(n)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "apiVersion")
}

func TestIsKRM_MissingKind(t *testing.T) {
	n := mustParse(t, `
apiVersion: v1
metadata:
  name: my-cm
`)
	err := IsKRM(n)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kind")
}

func TestIsKRM_MissingName(t *testing.T) {
	n := mustParse(t, `
apiVersion: v1
kind: ConfigMap
metadata: {}
`)
	err := IsKRM(n)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name")
}

func TestIsKRM_NoMetadataAtAll(t *testing.T) {
	n := mustParse(t, `
replicaCount: 3
image:
  repository: nginx
`)
	err := IsKRM(n)
	assert.Error(t, err)
}

// TestIsKRM_ListKindDoesNotRequireName verifies a nameless List-kind
// aggregate (e.g. ConfigMapList) is valid KRM.
func TestIsKRM_ListKindDoesNotRequireName(t *testing.T) {
	n := mustParse(t, `
apiVersion: v1
kind: ConfigMapList
items:
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: item-one
`)
	assert.NoError(t, IsKRM(n))
}

// TestIsKRM_ListKindStillRequiresAPIVersionAndKind verifies the List
// carve-out only waives the name requirement.
func TestIsKRM_ListKindStillRequiresAPIVersionAndKind(t *testing.T) {
	t.Run("missing apiVersion", func(t *testing.T) {
		n := mustParse(t, `
kind: ConfigMapList
items: []
`)
		err := IsKRM(n)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "apiVersion")
	})
	t.Run("missing kind entirely", func(t *testing.T) {
		n := mustParse(t, `
apiVersion: v1
items: []
`)
		err := IsKRM(n)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "kind")
	})
}

// TestIsKRM_GenerateNameExemptsFromName verifies a resource using
// metadata.generateName instead of metadata.name is valid KRM.
func TestIsKRM_GenerateNameExemptsFromName(t *testing.T) {
	n := mustParse(t, `
apiVersion: batch/v1
kind: Job
metadata:
  generateName: my-job-
`)
	assert.NoError(t, IsKRM(n))
}

// TestIsKRM_EmptyGenerateNameStillRequiresName verifies an explicitly
// empty generateName doesn't count as having one.
func TestIsKRM_EmptyGenerateNameStillRequiresName(t *testing.T) {
	n := mustParse(t, `
apiVersion: batch/v1
kind: Job
metadata:
  generateName: ""
`)
	err := IsKRM(n)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name")
}

// TestIsKRM_LocalConfigExemptsFromName verifies a nameless resource
// annotated as local-config (kpt's "config.kubernetes.io/local-config"
// convention) is valid KRM, matching pkg/live's own filterLocalConfig.
func TestIsKRM_LocalConfigExemptsFromName(t *testing.T) {
	n := mustParse(t, `
apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    config.kubernetes.io/local-config: "true"
`)
	assert.NoError(t, IsKRM(n))
}

// TestIsKRM_LocalConfigFalseStillRequiresName verifies an explicit
// "false" value (meaning "not local-config", per filterLocalConfig)
// does not exempt the resource from needing a name.
func TestIsKRM_LocalConfigFalseStillRequiresName(t *testing.T) {
	n := mustParse(t, `
apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    config.kubernetes.io/local-config: "false"
`)
	err := IsKRM(n)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name")
}

func TestIsKRM_KustomizationFileNameCarveOut(t *testing.T) {
	n := mustParse(t, `
resources:
- deployment.yaml
`)
	assert.NoError(t, n.PipeE(yaml.SetAnnotation("config.kubernetes.io/path", "overlays/dev/kustomization.yaml")))
	assert.NoError(t, IsKRM(n), "a file named kustomization.yaml must be exempt from KRM validation regardless of its shape")
}

func TestIsKRM_KustomizationAPIGroupCarveOut(t *testing.T) {
	n := mustParse(t, `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- deployment.yaml
`)
	assert.NoError(t, IsKRM(n))
}

func TestIsKRM_BareKustomizationKindNoAPIVersion(t *testing.T) {
	n := mustParse(t, `
kind: Kustomization
resources:
- deployment.yaml
`)
	assert.NoError(t, IsKRM(n))
}

func TestAreKRM_ReportsFirstFailureWithPath(t *testing.T) {
	valid := mustParse(t, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: good
`)
	assert.NoError(t, valid.PipeE(yaml.SetAnnotation("config.kubernetes.io/path", "good.yaml")))

	invalid := mustParse(t, `
replicaCount: 3
`)
	assert.NoError(t, invalid.PipeE(yaml.SetAnnotation("config.kubernetes.io/path", "bad.yaml")))

	err := AreKRM([]*yaml.RNode{valid, invalid})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad.yaml")
}

func TestAreKRM_AllValidSucceeds(t *testing.T) {
	a := mustParse(t, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: a
`)
	b := mustParse(t, `
apiVersion: v1
kind: ConfigMapList
items: []
`)
	assert.NoError(t, AreKRM([]*yaml.RNode{a, b}))
}
