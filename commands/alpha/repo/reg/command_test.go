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

package reg

import (
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fakeprinter "github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
)

var (
	update = flag.Bool("update", false, "update golden files")
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

type httpAction struct {
	method       string
	path         string
	wantRequest  string
	sendResponse string
}

type testcase struct {
	name    string
	args    []string
	actions []httpAction // http request to expect and responses to send back
}

func TestRepoReg(t *testing.T) {
	testdata, err := filepath.Abs(filepath.Join(".", "testdata"))
	if err != nil {
		t.Fatalf("Failed to find testdata: %v", err)
	}

	for _, tc := range []testcase{
		{
			name: "SimpleRegister",
			args: []string{"https://github.com/platkrm/test-blueprints"},
			actions: []httpAction{
				{
					method:       http.MethodPost,
					path:         "/apis/config.porch.kpt.dev/v1alpha1/namespaces/default/repositories",
					wantRequest:  "simple-repository.yaml",
					sendResponse: "simple-repository.yaml",
				},
			},
		},
		{
			name: "AuthRegister",
			args: []string{"https://github.com/platkrm/test-blueprints.git", "--repo-basic-username=test-username", "--repo-basic-password=test-password"},
			actions: []httpAction{
				{
					method:       http.MethodPost,
					path:         "/api/v1/namespaces/default/secrets",
					wantRequest:  "auth-secret.yaml",
					sendResponse: "auth-secret.yaml",
				},
				{
					method:       http.MethodPost,
					path:         "/apis/config.porch.kpt.dev/v1alpha1/namespaces/default/repositories",
					wantRequest:  "auth-repository.yaml",
					sendResponse: "auth-repository.yaml",
				},
			},
		},
		{
			name: "FullRegister",
			args: []string{
				"https://github.com/platkrm/test-blueprints.git",
				"--name=repository-resource-name",
				"--description=\"Test Repository Description\"",
				"--deployment",
				"--directory=/catalog",
				"--branch=main-branch",
				"--create-branch",
				"--namespace=repository-namespace",
			},
			actions: []httpAction{
				{
					method:       http.MethodPost,
					path:         "/apis/config.porch.kpt.dev/v1alpha1/namespaces/repository-namespace/repositories",
					wantRequest:  "full-repository.yaml",
					sendResponse: "full-repository.yaml",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Create fake Porch Server
			porch := createFakePorch(t, tc.actions, func(action httpAction, w http.ResponseWriter, r *http.Request) {
				// TODO: contents of this function is generic; move to shared utility in testutil.
				var requestBody []byte
				switch r.Header.Get("Content-Encoding") {
				case "":
					b, err := io.ReadAll(r.Body)
					if err != nil {
						t.Fatalf("Failed to read request body: %v", err)
					}
					requestBody = b

				default:
					t.Fatalf("unhandled content-encoding %q", r.Header.Get("Content-Encoding"))
				}

				var body interface{}
				switch r.Header.Get("Content-Type") {
				case "application/json":
					if err := json.Unmarshal(requestBody, &body); err != nil {
						t.Fatalf("Failed to unmarshal body: %v\n%s\n", err, string(requestBody))
					}

				// case "application/vnd.kubernetes.protobuf":
				// Proto encoding is not handled https://kubernetes.io/docs/reference/using-api/api-concepts/#protobuf-encoding

				default:
					t.Fatalf("unhandled content-type %q", r.Header.Get("Content-Type"))
				}

				wantFile := filepath.Join(testdata, action.wantRequest)

				if *update {
					data, err := yaml.Marshal(body)
					if err != nil {
						t.Fatalf("Failed to marshal request body as YAML: %v", err)
					}
					if err := os.WriteFile(wantFile, data, 0644); err != nil {
						t.Fatalf("Failed to update golden file %q: %v", wantFile, err)
					}
				}

				var want interface{}
				wantBytes, err := os.ReadFile(wantFile)
				if err != nil {
					t.Fatalf("Failed to reead golden file %q: %v", wantFile, err)
				}
				if err := yaml.Unmarshal(wantBytes, &want); err != nil {
					t.Fatalf("Failed to unmarshal expected body %q: %v", wantFile, err)
				}

				if !cmp.Equal(want, body) {
					t.Errorf("Unexpected request body for %q (-want, +got) %s", r.RequestURI, cmp.Diff(want, body))
				}

				respData, err := os.ReadFile(filepath.Join(testdata, action.sendResponse))
				if err != nil {
					t.Fatalf("Failed to read response file %q: %v", action.sendResponse, err)
				}
				var resp interface{}
				if err := yaml.Unmarshal(respData, &resp); err != nil {
					t.Fatalf("Failed to unmarshal desired response %q: %v", action.sendResponse, err)
				}
				respJSON, err := json.Marshal(resp)
				if err != nil {
					t.Fatalf("Failed to marshal response body as JSON: %v", err)
				}

				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write(respJSON); err != nil {
					t.Errorf("Failed to write resonse body %q: %v", action.sendResponse, err)
				}
			})

			// Create a test HTTP server.
			server := httptest.NewServer(porch)
			defer server.Close()

			// Create Kubeconfig
			url := server.URL
			usePersistentConfig := false
			rcg := genericclioptions.NewConfigFlags(usePersistentConfig)
			rcg.APIServer = &url
			rcg.WrapConfigFn = func(restConfig *rest.Config) *rest.Config {
				// Force use of JSON encoding
				restConfig.ContentType = "application/json"
				return restConfig
			}
			namespace := "default"
			rcg.Namespace = &namespace
			ctx := fakeprinter.CtxWithDefaultPrinter()

			cmd := NewCommand(ctx, rcg)
			rcg.AddFlags(cmd.PersistentFlags()) // Add global flags
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err != nil {
				t.Errorf("Executing repo register %s failed: %v", strings.Join(tc.args, " "), err)
			}
		})
	}
}

func createFakePorch(t *testing.T, actions []httpAction, handler func(action httpAction, w http.ResponseWriter, r *http.Request)) *fakePorch {
	actionMap := map[request]httpAction{}
	for _, a := range actions {
		actionMap[request{
			method: a.method,
			url:    a.path,
		}] = a
	}
	return &fakePorch{
		T:       t,
		actions: actionMap,
		handler: handler,
	}
}

// TODO: Move the below to shared testing utility
type request struct {
	method, url string
}

type fakePorch struct {
	*testing.T
	actions map[request]httpAction
	handler func(action httpAction, w http.ResponseWriter, r *http.Request)
}

var _ http.Handler = &fakePorch{}

func (p *fakePorch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.Logf("%s\n", r.RequestURI)
	action, ok := p.actions[request{method: r.Method, url: r.URL.Path}]
	if !ok {
		p.Logf("handler not found for method %q url %q", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	p.handler(action, w, r)
}
