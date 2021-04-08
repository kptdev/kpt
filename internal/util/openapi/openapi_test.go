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

package openapi

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest/fake"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"sigs.k8s.io/kustomize/kyaml/openapi"
)

func TestSomething(t *testing.T) {
	testCases := []struct {
		name                 string
		schemaSource         string
		schemaPath           string
		response             *http.Response
		includesRefString    string
		notIncludesRefString string
		expectError          bool
	}{
		{
			name:              "no schemaSource provided should lead to error",
			schemaSource:      "",
			includesRefString: "#/definitions/io.k8s.api.core.v1.PodSpec",
			expectError:       true,
		},
		{
			name:         "schemaSource cluster with successful fetch",
			schemaSource: "cluster",
			response: &http.Response{StatusCode: http.StatusOK,
				Header: cmdtesting.DefaultHeader(), Body: getSchema(t, "clusterschema.json")},
			includesRefString:    "#/definitions/io.k8s.clusterSchema",
			notIncludesRefString: "#/definitions/io.k8s.api.core.v1.PodSpec",
			expectError:          false,
		},
		{
			name:         "schemaSource cluster with failed fetch",
			schemaSource: "cluster",
			response: &http.Response{StatusCode: http.StatusNotFound,
				Header: cmdtesting.DefaultHeader(), Body: cmdtesting.StringBody("")},
			expectError: true,
		},
		{
			name:                 "schemaSource file with valid path",
			schemaSource:         "file",
			schemaPath:           "testdata/fileschema.json",
			includesRefString:    "#/definitions/io.k8s.fileSchema",
			notIncludesRefString: "#/definitions/io.k8s.api.core.v1.PodSpec",
			expectError:          false,
		},
		{
			name:         "schemaSource file with invalid path",
			schemaSource: "file",
			schemaPath:   "testdata/notfound.json",
			expectError:  true,
		},
		{
			name:              "schemaSource builtin",
			schemaSource:      "builtin",
			includesRefString: "#/definitions/io.k8s.api.core.v1.PodSpec",
			expectError:       false,
		},
		{
			name:         "unknown schemasource",
			schemaSource: "unknown",
			expectError:  true,
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace("test-namespace")
			defer tf.Cleanup()

			tf.ClientConfigVal.GroupVersion = &schema.GroupVersion{
				Group:   "",
				Version: "v1",
			}
			tf.ClientConfigVal.NegotiatedSerializer = resource.UnstructuredPlusDefaultContentConfig().NegotiatedSerializer

			tf.Client = &fake.RESTClient{
				NegotiatedSerializer: resource.UnstructuredPlusDefaultContentConfig().NegotiatedSerializer,
				Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
					if req.Method == http.MethodGet && req.URL.Path == "/openapi/v2" {
						return test.response, nil
					}
					t.Fatalf("unexpected request: %#v\n%#v", req.URL, req)
					return nil, nil
				}),
			}

			openapi.ResetOpenAPI()

			err := ConfigureOpenAPI(tf, test.schemaSource, test.schemaPath)
			if test.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			ref, err := spec.NewRef(test.includesRefString)
			assert.NoError(t, err)
			res, err := openapi.Resolve(&ref, openapi.Schema())
			assert.NoError(t, err)
			assert.NotNil(t, res)

			// If the notIncludesRefString is specified, make sure
			// the schema does not include the reference.
			if test.notIncludesRefString != "" {
				ref2, err := spec.NewRef(test.notIncludesRefString)
				assert.NoError(t, err)
				res2, _ := openapi.Resolve(&ref2, openapi.Schema())
				assert.Nil(t, res2)
			}

			// Verify that we have the Kustomize openAPI included.
			kustomizeRef, _ := spec.NewRef("#/definitions/io.k8s.api.apps.v1.Kustomization")
			kustomizeRes, err := openapi.Resolve(&kustomizeRef, openapi.Schema())
			assert.NoError(t, err)
			assert.NotNil(t, kustomizeRes)
		})
	}
}

func getSchema(t *testing.T, filename string) io.ReadCloser {
	b, err := ioutil.ReadFile("testdata/" + filename)
	if err != nil {
		t.Fatal(err)
	}
	return ioutil.NopCloser(bytes.NewBuffer(b))
}
