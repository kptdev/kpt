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

package runtime

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/kptdev/kpt/pkg/lib/runneroptions"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinesToSlice(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected []string
	}{
		"empty": {
			input:    "",
			expected: nil,
		},
		"whitespace only": {
			input:    "  \t\n  ",
			expected: nil,
		},
		"single line": {
			input:    "v1.0.0",
			expected: []string{"v1.0.0"},
		},
		"single line with surrounding space": {
			input:    "  v1.0.0  \n",
			expected: []string{"v1.0.0"},
		},
		"multiple lines": {
			input:    "v1.0.0\nv1.0.1\nv1.0.2",
			expected: []string{"v1.0.0", "v1.0.1", "v1.0.2"},
		},
		"windows line endings": {
			input:    "a\r\nb\r\nc",
			expected: []string{"a", "b", "c"},
		},
		"blank line not preserved": {
			input:    "a\n\nb",
			expected: []string{"a", "b"},
		},
		"per-line trim": {
			input:    "  foo  \n  bar  ",
			expected: []string{"foo", "bar"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := linesToSlice(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestRegClientLister_Name(t *testing.T) {
	l := &RegClientLister{}
	assert.Equal(t, "regclient", l.Name())
}

func TestLocalLister_Name(t *testing.T) {
	tests := map[string]struct {
		binary   string
		expected string
	}{
		"docker": {
			binary:   "docker",
			expected: "local-docker",
		},
		"empty binary": {
			binary:   "",
			expected: "local-",
		},
		"path-like binary": {
			binary:   "/usr/local/bin/nerdctl",
			expected: "local-/usr/local/bin/nerdctl",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			l := &LocalLister{Binary: tc.binary}
			assert.Equal(t, tc.expected, l.Name())
		})
	}
}

func TestRegClientLister_Authentication(t *testing.T) {
	const expectedUser = "testuser"
	const expectedPass = "testsecret"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != expectedUser || p != expectedPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.URL.Path == "/v2/myorg/myfn/tags/list" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"myorg/myfn","tags":["v1.0.0","v1.1.0","v2.0.0"]}`))
			return
		}

		http.NotFound(w, r)
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	require.NoError(t, err)
	host := u.Host // 127.0.0.1:port

	ctx := t.Context()
	image := host + "/myorg/myfn"

	// 1. Without credentials -> listing tags should fail with 401 Unauthorized
	chNoAuth := config.HostNewName(host)
	chNoAuth.TLS = config.TLSDisabled
	rcOptsNoAuth := []regclient.Opt{
		regclient.WithUserAgent(UserAgent),
		regclient.WithConfigHost(*chNoAuth),
	}

	listerNoAuth := &RegClientLister{
		Client: regclient.New(rcOptsNoAuth...),
	}
	_, err = listerNoAuth.List(ctx, image)
	require.Error(t, err, "expected authentication error when credentials are not provided")

	// 2. With credentials via RegistryCredentials -> listing tags should succeed
	creds := runneroptions.RegistryCredentials{
		Hosts: []runneroptions.RegistryHost{
			{
				Name:     host,
				Username: expectedUser,
				Password: expectedPass,
				Insecure: true,
			},
		},
	}
	hosts, err := creds.ToRegClientHosts()
	require.NoError(t, err)

	// Since httptest.Server is plain HTTP, set TLSDisabled
	hosts[0].TLS = config.TLSDisabled

	rcOptsAuth := []regclient.Opt{
		regclient.WithUserAgent(UserAgent),
		regclient.WithConfigHost(hosts...),
	}

	listerAuth := &RegClientLister{
		Client: regclient.New(rcOptsAuth...),
	}
	tags, err := listerAuth.List(ctx, image)
	require.NoError(t, err)
	assert.Equal(t, []string{"v1.0.0", "v1.1.0", "v2.0.0"}, tags)
}
