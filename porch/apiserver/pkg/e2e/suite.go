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

package e2e

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"gopkg.in/yaml.v2"
	coreapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	// TODO: accept a flag?
	PorchTestConfigFile = "porch-test-config.yaml"
)

// Format of the optional test configuration file to enable running
// the test with specified GCP project, repositories and authentication.
type PorchTestConfig struct {
	Project string `json:"project"`
	Git     struct {
		Repo      string   `json:"repo"`
		Branch    string   `json:"branch"`
		Directory string   `json:"directory"`
		Username  string   `json:"username"`
		Token     Password `json:"token"`
	} `json:"git"`
	Oci struct {
		Registry string `json:"registry"`
	} `yaml:"oci"`
}

type Password string

func (p Password) String() string {
	return "*************"
}

type TestSuite struct {
	*testing.T
	client client.Client

	// Namespace used for tests
	namespace string
	ptc       PorchTestConfig
}

type Initializer interface {
	Initialize(ctx context.Context, t *testing.T)
}

var _ Initializer = &TestSuite{}

func (pt *TestSuite) Initialize(ctx context.Context, t *testing.T) {
	pt.ptc = readTestConfig(t)

	cfg, err := config.GetConfig()
	if err != nil {
		t.Skipf("Skipping tests - cannot obtain k8s client config: %v", err)
	}

	t.Logf("Testing against server: %q", cfg.Host)
	cfg.UserAgent = "Porch Test"

	scheme := runtime.NewScheme()
	if err := porchapi.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to initialize Porch API client: %v", err)
	}
	if err := configapi.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to initialize Config API client: %v", err)
	}
	if err := coreapi.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to initialize Core API client: %v", err)
	}
	if c, err := client.New(cfg, client.Options{
		Scheme: scheme,
	}); err != nil {
		t.Fatalf("Failed to initialize k8s client: %v", err)
	} else {
		pt.client = c
	}

	namespace := fmt.Sprintf("porch-test-%d", time.Now().UnixMicro())
	if err := pt.client.Create(ctx, &coreapi.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}); err != nil {
		t.Fatalf("Failed to create test namespace %q: %v", namespace, err)
	} else {
		pt.namespace = namespace
		c := pt.client
		t.Cleanup(func() {
			if err := c.Delete(ctx, &coreapi.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}); err != nil {
				t.Errorf("Failed to clean up namespace %q: %v", namespace, err)
			} else {
				t.Logf("Successfully cleaned up namespace %q", namespace)
			}
		})
	}
}

type ErrorHandler func(format string, args ...interface{})

func (t *TestSuite) get(ctx context.Context, key client.ObjectKey, obj client.Object, eh ErrorHandler) {
	if err := t.client.Get(ctx, key, obj); err != nil {
		eh("failed to get resource %s %s/%s: %v", obj.GetObjectKind().GroupVersionKind(), key.Name, key.Namespace, err)
	}
}

func (c *TestSuite) list(ctx context.Context, list client.ObjectList, opts []client.ListOption, eh ErrorHandler) {
	if err := c.client.List(ctx, list, opts...); err != nil {
		eh("failed to list resources %s %+v: %v", list.GetObjectKind().GroupVersionKind(), list, err)
	}
}

func (c *TestSuite) create(ctx context.Context, obj client.Object, opts []client.CreateOption, eh ErrorHandler) {
	if err := c.client.Create(ctx, obj, opts...); err != nil {
		eh("failed to create resource %s %s/%s: %v", obj.GetObjectKind().GroupVersionKind(), obj.GetNamespace(), obj.GetName(), err)
	}
}

func (c *TestSuite) delete(ctx context.Context, obj client.Object, opts []client.DeleteOption, eh ErrorHandler) {
	if err := c.client.Delete(ctx, obj, opts...); err != nil {
		eh("failed to delete resource %s %s/%s: %v", obj.GetObjectKind().GroupVersionKind(), obj.GetNamespace(), obj.GetName(), err)
	}
}

// update(ctx context.Context, obj Object, opts ...UpdateOption) error
// patch(ctx context.Context, obj Object, patch Patch, opts ...PatchOption) error
// deleteAllOf(ctx context.Context, obj Object, opts ...DeleteAllOfOption) error

func (c *TestSuite) GetE(ctx context.Context, key client.ObjectKey, obj client.Object) {
	c.get(ctx, key, obj, ErrorHandler(c.Errorf))
}

func (c *TestSuite) ListE(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
	c.list(ctx, list, opts, c.Errorf)
}

func (c *TestSuite) CreateF(ctx context.Context, obj client.Object, opts ...client.CreateOption) {
	c.create(ctx, obj, opts, c.Fatalf)
}

func (c *TestSuite) CreateE(ctx context.Context, obj client.Object, opts ...client.CreateOption) {
	c.create(ctx, obj, opts, c.Errorf)
}
func (c *TestSuite) DeleteE(ctx context.Context, obj client.Object, opts ...client.DeleteOption) {
	c.delete(ctx, obj, opts, c.Errorf)
}

func (c *TestSuite) DeleteL(ctx context.Context, obj client.Object, opts ...client.DeleteOption) {
	c.delete(ctx, obj, opts, c.Logf)
}

// Update(ctx context.Context, obj Object, opts ...UpdateOption) error
// Patch(ctx context.Context, obj Object, patch Patch, opts ...PatchOption) error
// DeleteAllOf(ctx context.Context, obj Object, opts ...DeleteAllOfOption) error

func readTestConfig(t *testing.T) PorchTestConfig {
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Logf("Cannot get user config directory: %v; proceeding without config", err)
		return PorchTestConfig{}
	}
	path := filepath.Join(dir, PorchTestConfigFile)
	config, err := ioutil.ReadFile(path)
	if err != nil {
		t.Logf("Cannot read Porch test config %q: %v; proceeding without config", path, err)
		return PorchTestConfig{}
	}
	var ptc PorchTestConfig
	if err := yaml.Unmarshal(config, &ptc); err != nil {
		t.Fatalf("Failed to parse Porch test config %q: %v; failing...", path, err)
		return PorchTestConfig{}
	}
	return ptc
}
