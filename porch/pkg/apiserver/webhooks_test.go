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

package apiserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateCerts(t *testing.T) {
	dir := t.TempDir()
	defer func() {
		require.NoError(t, os.RemoveAll(dir))
	}()

	caCert, err := createCerts(dir)
	require.NoError(t, err)

	caStr := strings.TrimSpace(string(caCert))
	require.True(t, strings.HasPrefix(caStr, "-----BEGIN CERTIFICATE-----\n"))
	require.True(t, strings.HasSuffix(caStr, "\n-----END CERTIFICATE-----"))

	crt, err := os.ReadFile(filepath.Join(dir, "tls.crt"))
	require.NoError(t, err)

	key, err := os.ReadFile(filepath.Join(dir, "tls.key"))
	require.NoError(t, err)

	crtStr := strings.TrimSpace(string(crt))
	require.True(t, strings.HasPrefix(crtStr, "-----BEGIN CERTIFICATE-----\n"))
	require.True(t, strings.HasSuffix(crtStr, "\n-----END CERTIFICATE-----"))

	keyStr := strings.TrimSpace(string(key))
	require.True(t, strings.HasPrefix(keyStr, "-----BEGIN RSA PRIVATE KEY-----\n"))
	require.True(t, strings.HasSuffix(keyStr, "\n-----END RSA PRIVATE KEY-----"))
}

func TestValidateDeletion(t *testing.T) {
	t.Run("invalid content-type", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, serverEndpoint, nil)
		require.NoError(t, err)
		request.Header.Set("Content-Type", "foo")
		response := httptest.NewRecorder()

		validateDeletion(response, request)
		require.Equal(t,
			"error getting admission review from request: expected Content-Type 'application/json'",
			response.Body.String())
	})
	t.Run("valid content-type, but no body", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, serverEndpoint, nil)
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		validateDeletion(response, request)
		require.Equal(t,
			"error getting admission review from request: admission review request is empty",
			response.Body.String())
	})
	t.Run("wrong GVK in request", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodPost, serverEndpoint, nil)
		require.NoError(t, err)

		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		admissionReviewRequest := admissionv1.AdmissionReview{
			TypeMeta: v1.TypeMeta{
				Kind:       "AdmissionReview",
				APIVersion: "admission.k8s.io/v1",
			},
			Request: &admissionv1.AdmissionRequest{
				Resource: v1.GroupVersionResource{
					Group:    "porch.kpt.dev",
					Version:  "v1alpha1",
					Resource: "not-a-package-revision",
				},
			},
		}

		body, err := json.Marshal(admissionReviewRequest)
		require.NoError(t, err)

		request.Body = io.NopCloser(bytes.NewReader(body))
		validateDeletion(response, request)
		require.Equal(t,
			"did not receive PackageRevision, got not-a-package-revision",
			response.Body.String())
	})
}
