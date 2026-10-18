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

package runneroptions

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regclient/regclient/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryHost_ToConfigHost(t *testing.T) {
	h := RegistryHost{
		Name:       "example.com",
		Username:   "testuser",
		Password:   "testpass",
		Token:      "testtoken",
		CredHelper: "docker-credential-gcr",
		Insecure:   true,
	}

	ch := h.ToConfigHost()
	assert.Equal(t, "example.com", ch.Name)
	assert.Equal(t, "testuser", ch.User)
	assert.Equal(t, "testpass", ch.Pass)
	assert.Equal(t, "testtoken", ch.Token)
	assert.Equal(t, "docker-credential-gcr", ch.CredHelper)
	assert.Equal(t, config.TLSInsecure, ch.TLS)
}

func TestRegistryHost_SecurityRedaction(t *testing.T) {
	h := RegistryHost{
		Name:     "private.registry.io",
		Username: "admin",
		Password: "super-secret-password-123",
		Token:    "secret-bearer-token-456",
		Insecure: false,
	}

	// Verify fmt.Sprintf %s / %v / %+v / %#v does not leak password or token
	str := fmt.Sprintf("%v", h)
	assert.Contains(t, str, "private.registry.io")
	assert.Contains(t, str, "user:admin")
	assert.Contains(t, str, "pass:[REDACTED]")
	assert.Contains(t, str, "token:[REDACTED]")
	assert.NotContains(t, str, "super-secret-password-123")
	assert.NotContains(t, str, "secret-bearer-token-456")

	goStr := fmt.Sprintf("%#v", h)
	assert.NotContains(t, goStr, "super-secret-password-123")
	assert.NotContains(t, goStr, "secret-bearer-token-456")

	// Verify JSON marshaling redacts sensitive fields
	data, err := json.Marshal(h)
	require.NoError(t, err)
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `[REDACTED]`)
	assert.NotContains(t, jsonStr, "super-secret-password-123")
	assert.NotContains(t, jsonStr, "secret-bearer-token-456")
}

func TestRegistryCredentials_SecurityRedaction(t *testing.T) {
	rawDockerJSON := []byte(`{"auths":{"ghcr.io":{"username":"foo","password":"secretpassword"}}}`)
	creds := RegistryCredentials{
		DockerConfigFile: "/etc/docker/config.json",
		DockerConfigJSON: rawDockerJSON,
		Hosts: []RegistryHost{
			{
				Name:     "registry.example.com",
				Username: "user1",
				Password: "my-password",
				Token:    "my-token",
			},
		},
		RegClientHosts: []config.Host{
			{
				Name: "regclient.example.com",
				User: "reguser",
				Pass: "regpass",
			},
		},
		DisableDefaultDockerCreds: true,
	}

	// Verify String / GoString redaction
	str := fmt.Sprintf("%+v", creds)
	assert.Contains(t, str, "DockerConfigFile:\"/etc/docker/config.json\"")
	assert.Contains(t, str, fmt.Sprintf("DockerConfigJSON:[%d bytes, [REDACTED]]", len(rawDockerJSON)))
	assert.Contains(t, str, "pass:[REDACTED]")
	assert.Contains(t, str, "token:[REDACTED]")
	assert.Contains(t, str, "Pass:[REDACTED]")
	assert.Contains(t, str, "DisableDefaultDockerCreds:true")
	assert.NotContains(t, str, "secretpassword")
	assert.NotContains(t, str, "my-password")
	assert.NotContains(t, str, "my-token")
	assert.NotContains(t, str, "regpass")

	// Verify JSON marshaling redaction
	data, err := json.Marshal(creds)
	require.NoError(t, err)
	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "secretpassword")
	assert.NotContains(t, jsonStr, "my-password")
	assert.NotContains(t, jsonStr, "my-token")
	assert.NotContains(t, jsonStr, "regpass")
	assert.Contains(t, jsonStr, `[REDACTED]`)
}

func TestRegistryCredentials_IsZero(t *testing.T) {
	var empty RegistryCredentials
	assert.True(t, empty.IsZero())

	assert.False(t, RegistryCredentials{DockerConfigFile: "path"}.IsZero())
	assert.False(t, RegistryCredentials{DockerConfigJSON: []byte("data")}.IsZero())
	assert.False(t, RegistryCredentials{Hosts: []RegistryHost{{Name: "host"}}}.IsZero())
	assert.False(t, RegistryCredentials{RegClientHosts: []config.Host{{Name: "host"}}}.IsZero())
	assert.False(t, RegistryCredentials{DisableDefaultDockerCreds: true}.IsZero())
}

func TestParseDockerConfigJSON_Standard(t *testing.T) {
	authStr := base64.StdEncoding.EncodeToString([]byte("alice:secretalice"))
	dockerJSON := fmt.Sprintf(`{
		"auths": {
			"ghcr.io": {
				"auth": "%s"
			},
			"https://index.docker.io/v1/": {
				"username": "bob",
				"password": "secretbob"
			},
			"gcr.io": {
				"identitytoken": "gcr-token"
			}
		},
		"credHelpers": {
			"us-central1-docker.pkg.dev": "google-cloud"
		}
	}`, authStr)

	hosts, err := parseDockerConfigJSON([]byte(dockerJSON), true)
	require.NoError(t, err)
	require.Len(t, hosts, 4)

	hostMap := map[string]config.Host{}
	for _, h := range hosts {
		hostMap[h.Name] = h
	}

	ghcr, ok := hostMap["ghcr.io"]
	require.True(t, ok)
	assert.Equal(t, "alice", ghcr.User)
	assert.Equal(t, "secretalice", ghcr.Pass)

	docker, ok := hostMap["docker.io"]
	require.True(t, ok)
	assert.Equal(t, "bob", docker.User)
	assert.Equal(t, "secretbob", docker.Pass)

	gcr, ok := hostMap["gcr.io"]
	require.True(t, ok)
	assert.Equal(t, "gcr-token", gcr.Token)

	gar, ok := hostMap["us-central1-docker.pkg.dev"]
	require.True(t, ok)
	assert.Equal(t, "docker-credential-google-cloud", gar.CredHelper)
}

func TestParseDockerConfigJSON_LegacyDockercfg(t *testing.T) {
	authStr := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	dockercfg := fmt.Sprintf(`{
		"myregistry.corp.net:5000": {
			"auth": "%s",
			"email": "user@corp.net"
		}
	}`, authStr)

	hosts, err := parseDockerConfigJSON([]byte(dockercfg), false)
	require.NoError(t, err)
	require.Len(t, hosts, 1)

	assert.Equal(t, "myregistry.corp.net:5000", hosts[0].Name)
	assert.Equal(t, "user", hosts[0].User)
	assert.Equal(t, "pass", hosts[0].Pass)
}

func TestParseDockerConfigJSON_EmptyAndInvalid(t *testing.T) {
	// Empty or whitespace
	hosts, err := parseDockerConfigJSON([]byte(""), false)
	require.NoError(t, err)
	assert.Nil(t, hosts)

	hosts, err = parseDockerConfigJSON([]byte("   \n\t  "), false)
	require.NoError(t, err)
	assert.Nil(t, hosts)

	hosts, err = parseDockerConfigJSON([]byte("{}"), false)
	require.NoError(t, err)
	assert.Nil(t, hosts)

	// Present-but-empty auths map (common default Docker configs)
	hosts, err = parseDockerConfigJSON([]byte(`{"auths":{}}`), false)
	require.NoError(t, err)
	assert.Nil(t, hosts)

	hosts, err = parseDockerConfigJSON([]byte(`{"auths":{},"currentContext":"default"}`), false)
	require.NoError(t, err)
	assert.Nil(t, hosts)

	// Store-only config without specific registry hosts but AllowCredentialHelpers enabled
	hosts, err = parseDockerConfigJSON([]byte(`{"credsStore":"pass"}`), true)
	require.NoError(t, err)
	assert.Nil(t, hosts)

	// Invalid JSON
	_, err = parseDockerConfigJSON([]byte("not valid json"), false)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid docker config JSON format"))
}

func TestParseDockerConfigJSON_SecurityHelperDirectives(t *testing.T) {
	t.Run("rejects credHelpers in raw JSON by default", func(t *testing.T) {
		jsonStr := `{"credHelpers":{"gcr.io":"gcr"}}`
		_, err := parseDockerConfigJSON([]byte(jsonStr), false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "credential helper directives (credHelpers/credsStore) in DockerConfigJSON are not allowed by default")
	})

	t.Run("rejects credsStore in raw JSON by default", func(t *testing.T) {
		jsonStr := `{"auths":{"private.example.com":{}},"credsStore":"pass"}`
		_, err := parseDockerConfigJSON([]byte(jsonStr), false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "credential helper directives (credHelpers/credsStore) in DockerConfigJSON are not allowed by default")
	})

	t.Run("multi-tenant isolation enforces helper rejection with DisableDefaultDockerCreds", func(t *testing.T) {
		creds := RegistryCredentials{
			DisableDefaultDockerCreds: true,
			DockerConfigJSON:          []byte(`{"credHelpers":{"private.example.com":"gcr"}}`),
			AllowCredentialHelpers:    false,
		}
		_, err := creds.ToRegClientHosts()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "credential helper directives (credHelpers/credsStore) in DockerConfigJSON are not allowed by default")
	})

	t.Run("permits regular auths with DisableDefaultDockerCreds", func(t *testing.T) {
		creds := RegistryCredentials{
			DisableDefaultDockerCreds: true,
			DockerConfigJSON:          []byte(`{"auths":{"private.example.com":{"username":"u","password":"p"}}}`),
		}
		hosts, err := creds.ToRegClientHosts()
		require.NoError(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, "private.example.com", hosts[0].Name)
		assert.Equal(t, "u", hosts[0].User)
		assert.Equal(t, "p", hosts[0].Pass)
	})
}

func TestParseDockerConfigJSON_CredsStoreAndHelpers(t *testing.T) {
	t.Run("credsStore fallback when registry has no explicit credentials", func(t *testing.T) {
		jsonStr := `{"auths":{"private.example.com":{}},"credsStore":"pass"}`
		hosts, err := parseDockerConfigJSON([]byte(jsonStr), true)
		require.NoError(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, "private.example.com", hosts[0].Name)
		assert.Equal(t, "docker-credential-pass", hosts[0].CredHelper)
	})

	t.Run("per-registry credHelpers takes precedence over credsStore", func(t *testing.T) {
		jsonStr := `{
			"auths": {
				"private.example.com": {},
				"custom.example.com": {}
			},
			"credHelpers": {
				"custom.example.com": "custom"
			},
			"credsStore": "pass"
		}`
		hosts, err := parseDockerConfigJSON([]byte(jsonStr), true)
		require.NoError(t, err)
		require.Len(t, hosts, 2)

		hostMap := map[string]config.Host{}
		for _, h := range hosts {
			hostMap[h.Name] = h
		}

		assert.Equal(t, "docker-credential-pass", hostMap["private.example.com"].CredHelper)
		assert.Equal(t, "docker-credential-custom", hostMap["custom.example.com"].CredHelper)
	})

	t.Run("both auths and credHelpers contain index.docker.io", func(t *testing.T) {
		jsonStr := `{
			"auths": {
				"index.docker.io": {}
			},
			"credHelpers": {
				"index.docker.io": "desktop"
			}
		}`
		hosts, err := parseDockerConfigJSON([]byte(jsonStr), true)
		require.NoError(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, config.DockerRegistry, hosts[0].Name)
		assert.Equal(t, "docker-credential-desktop", hosts[0].CredHelper)
	})

	t.Run("credHelpers for registry not listed in auths", func(t *testing.T) {
		jsonStr := `{
			"auths": {
				"main.example.com": {
					"username": "user",
					"password": "pwd"
				}
			},
			"credHelpers": {
				"standalone.example.com": "ecr-login"
			}
		}`
		hosts, err := parseDockerConfigJSON([]byte(jsonStr), true)
		require.NoError(t, err)
		require.Len(t, hosts, 2)

		hostMap := map[string]config.Host{}
		for _, h := range hosts {
			hostMap[h.Name] = h
		}

		assert.Equal(t, "user", hostMap["main.example.com"].User)
		assert.Equal(t, "pwd", hostMap["main.example.com"].Pass)
		assert.Equal(t, "docker-credential-ecr-login", hostMap["standalone.example.com"].CredHelper)
	})

	t.Run("auths has v1 docker URL and credHelpers has index.docker.io", func(t *testing.T) {
		jsonStr := `{
			"auths": {
				"https://index.docker.io/v1/": {}
			},
			"credHelpers": {
				"index.docker.io": "desktop"
			}
		}`
		hosts, err := parseDockerConfigJSON([]byte(jsonStr), true)
		require.NoError(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, config.DockerRegistry, hosts[0].Name)
		assert.Equal(t, "docker-credential-desktop", hosts[0].CredHelper)
	})

	t.Run("helper with existing docker-credential- prefix is not doubled", func(t *testing.T) {
		jsonStr := `{
			"credHelpers": {
				"gcr.io": "docker-credential-gcr"
			}
		}`
		hosts, err := parseDockerConfigJSON([]byte(jsonStr), true)
		require.NoError(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, "gcr.io", hosts[0].Name)
		assert.Equal(t, "docker-credential-gcr", hosts[0].CredHelper)
	})

	t.Run("store-only Docker config discovers credentials via helper list", func(t *testing.T) {
		tempDir := t.TempDir()
		helperScript := filepath.Join(tempDir, "docker-credential-mockstore")
		scriptContent := "#!/bin/sh\n" +
			"if [ \"$1\" = \"list\" ]; then\n" +
			"  echo '{\"https://store.example.com\": \"storeuser\", \"index.docker.io\": \"dockeruser\"}'\n" +
			"  exit 0\n" +
			"fi\n" +
			"exit 1\n"
		err := os.WriteFile(helperScript, []byte(scriptContent), 0o755)
		require.NoError(t, err)

		origPath := os.Getenv("PATH")
		t.Setenv("PATH", fmt.Sprintf("%s%c%s", tempDir, os.PathListSeparator, origPath))

		jsonStr := `{"credsStore": "mockstore"}`
		hosts, err := parseDockerConfigJSON([]byte(jsonStr), true)
		require.NoError(t, err)
		require.Len(t, hosts, 2)

		hostMap := map[string]config.Host{}
		for _, h := range hosts {
			hostMap[h.Name] = h
		}

		assert.Equal(t, "storeuser", hostMap["store.example.com"].User)
		assert.Equal(t, "docker-credential-mockstore", hostMap["store.example.com"].CredHelper)
		assert.Equal(t, "dockeruser", hostMap[config.DockerRegistry].User)
		assert.Equal(t, "docker-credential-mockstore", hostMap[config.DockerRegistry].CredHelper)
	})
}

func TestRegistryCredentials_ToRegClientHosts(t *testing.T) {
	creds := RegistryCredentials{
		DockerConfigJSON: []byte(`{"auths":{"quay.io":{"username":"quayuser","password":"quaypassword"}}}`),
		Hosts: []RegistryHost{
			{
				Name:     "custom.registry.com",
				Username: "customuser",
				Password: "custompass",
			},
		},
		RegClientHosts: []config.Host{
			{
				Name: "advanced.registry.com",
				User: "advuser",
				Pass: "advpass",
			},
		},
	}

	hosts, err := creds.ToRegClientHosts()
	require.NoError(t, err)
	require.Len(t, hosts, 3)

	names := []string{hosts[0].Name, hosts[1].Name, hosts[2].Name}
	assert.Contains(t, names, "quay.io")
	assert.Contains(t, names, "custom.registry.com")
	assert.Contains(t, names, "advanced.registry.com")
}
