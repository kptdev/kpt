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
	"os/exec"
	"strings"

	"github.com/regclient/regclient/config"
)

const (
	redacted           = "[REDACTED]"
	dockerHelperPrefix = "docker-credential-"
)

// RegistryHost defines authentication and connection configuration for a single OCI registry host.
type RegistryHost struct {
	// Name is the registry hostname (e.g. "ghcr.io", "gcr.io", "my-registry.example.com").
	Name string `json:"name"`

	// Username for authentication.
	Username string `json:"username,omitempty"`

	// Password for authentication.
	Password string `json:"password,omitempty"`

	// Token for bearer token authentication.
	Token string `json:"token,omitempty"`

	// CredHelper is an optional credential helper command.
	CredHelper string `json:"credHelper,omitempty"`

	// Insecure disables TLS certificate verification for HTTPS connections.
	// For plain HTTP, use RegistryCredentials.RegClientHosts with config.TLSDisabled.
	Insecure bool `json:"insecure,omitempty"`
}

// ToConfigHost converts RegistryHost to a regclient config.Host.
func (h RegistryHost) ToConfigHost() config.Host {
	ch := config.HostNewName(h.Name)
	if h.Username != "" {
		ch.User = h.Username
	}
	if h.Password != "" {
		ch.Pass = h.Password
	}
	if h.Token != "" {
		ch.Token = h.Token
	}
	if h.CredHelper != "" {
		ch.CredHelper = h.CredHelper
	}
	if h.Insecure {
		ch.TLS = config.TLSInsecure
	}
	return *ch
}

func (h RegistryHost) String() string {
	var parts []string
	if h.Username != "" {
		parts = append(parts, fmt.Sprintf("user:%s", h.Username))
	}
	if h.Password != "" {
		parts = append(parts, "pass:"+redacted)
	}
	if h.Token != "" {
		parts = append(parts, "token:"+redacted)
	}
	if h.CredHelper != "" {
		parts = append(parts, fmt.Sprintf("helper:%s", h.CredHelper))
	}
	if h.Insecure {
		parts = append(parts, "insecure:true")
	}
	if len(parts) == 0 {
		return fmt.Sprintf("RegistryHost{Name:%q}", h.Name)
	}
	return fmt.Sprintf("RegistryHost{Name:%q, %s}", h.Name, strings.Join(parts, ", "))
}

func (h RegistryHost) GoString() string {
	return h.String()
}

func (h RegistryHost) MarshalJSON() ([]byte, error) {
	type Alias RegistryHost
	aux := Alias(h)
	if aux.Password != "" {
		aux.Password = redacted
	}
	if aux.Token != "" {
		aux.Token = redacted
	}
	return json.Marshal(aux)
}

// RegistryCredentials contains credentials and configuration for accessing private container registries.
// It supports multiple sources of credentials (files, raw JSON secrets, and structured hosts)
// and implements redacting stringers to prevent sensitive credentials from leaking into logs or errors.
type RegistryCredentials struct {
	// DockerConfigFile is an optional path to a Docker config.json file with registry logins.
	DockerConfigFile string `json:"dockerConfigFile,omitempty"`

	// DockerConfigJSON contains optional raw Docker configuration JSON bytes (e.g. from .dockerconfigjson Kubernetes secrets).
	DockerConfigJSON []byte `json:"dockerConfigJSON,omitempty"`

	// Hosts contains per-host registry configurations.
	Hosts []RegistryHost `json:"hosts,omitempty"`

	// RegClientHosts contains raw regclient config.Host entries for advanced callers.
	RegClientHosts []config.Host `json:"regClientHosts,omitempty"`

	// DisableDefaultDockerCreds disables loading credentials from the default Docker config file (~/.docker/config.json).
	// When true, only explicitly provided credentials will be used, providing multi-tenant isolation.
	DisableDefaultDockerCreds bool `json:"disableDefaultDockerCreds,omitempty"`

	// AllowCredentialHelpers permits executing external credential helpers (credHelpers and credsStore)
	// specified in DockerConfigJSON. By default, this is false to prevent untrusted tenant configurations
	// from invoking host binaries or accessing evaluator-local credentials in multi-tenant environments.
	AllowCredentialHelpers bool `json:"allowCredentialHelpers,omitempty"`
}

// IsZero returns true if no credential options or flags have been configured.
func (rc RegistryCredentials) IsZero() bool {
	return rc.DockerConfigFile == "" &&
		len(rc.DockerConfigJSON) == 0 &&
		len(rc.Hosts) == 0 &&
		len(rc.RegClientHosts) == 0 &&
		!rc.DisableDefaultDockerCreds &&
		!rc.AllowCredentialHelpers
}

func (rc RegistryCredentials) String() string {
	var parts []string
	if rc.DockerConfigFile != "" {
		parts = append(parts, fmt.Sprintf("DockerConfigFile:%q", rc.DockerConfigFile))
	}
	if len(rc.DockerConfigJSON) > 0 {
		parts = append(parts, fmt.Sprintf("DockerConfigJSON:[%d bytes, %s]", len(rc.DockerConfigJSON), redacted))
	}
	if len(rc.Hosts) > 0 {
		var hostStrs []string
		for _, h := range rc.Hosts {
			hostStrs = append(hostStrs, h.String())
		}
		parts = append(parts, fmt.Sprintf("Hosts:[%s]", strings.Join(hostStrs, ", ")))
	}
	if len(rc.RegClientHosts) > 0 {
		var regHostStrs []string
		for _, h := range rc.RegClientHosts {
			regHostStrs = append(regHostStrs, fmt.Sprintf("Host{Name:%q, User:%q, Pass:%s}", h.Name, h.User, redacted))
		}
		parts = append(parts, fmt.Sprintf("RegClientHosts:[%s]", strings.Join(regHostStrs, ", ")))
	}
	if rc.DisableDefaultDockerCreds {
		parts = append(parts, "DisableDefaultDockerCreds:true")
	}
	if rc.AllowCredentialHelpers {
		parts = append(parts, "AllowCredentialHelpers:true")
	}
	return fmt.Sprintf("RegistryCredentials{%s}", strings.Join(parts, ", "))
}

func (rc RegistryCredentials) GoString() string {
	return rc.String()
}

func (rc RegistryCredentials) MarshalJSON() ([]byte, error) {
	type Alias RegistryCredentials
	aux := struct {
		Alias
		DockerConfigJSON string `json:"dockerConfigJSON,omitempty"`
	}{
		Alias: Alias(rc),
	}
	if len(rc.DockerConfigJSON) > 0 {
		aux.DockerConfigJSON = redacted
	}
	if len(aux.RegClientHosts) > 0 {
		redactedRegHosts := make([]config.Host, len(aux.RegClientHosts))
		for i, h := range aux.RegClientHosts {
			redactedRegHosts[i] = h
			if redactedRegHosts[i].Pass != "" {
				redactedRegHosts[i].Pass = redacted
			}
			if redactedRegHosts[i].Token != "" {
				redactedRegHosts[i].Token = redacted
			}
			if redactedRegHosts[i].ClientKey != "" {
				redactedRegHosts[i].ClientKey = redacted
			}
		}
		aux.RegClientHosts = redactedRegHosts
	}
	return json.Marshal(aux)
}

// ToRegClientHosts parses and converts all in-memory credentials (Hosts, DockerConfigJSON, and RegClientHosts)
// into a slice of regclient config.Host entries ready to be registered with regclient.WithConfigHost.
func (rc RegistryCredentials) ToRegClientHosts() ([]config.Host, error) {
	var result []config.Host

	if len(rc.DockerConfigJSON) > 0 {
		parsedHosts, err := parseDockerConfigJSON(rc.DockerConfigJSON, rc.AllowCredentialHelpers)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DockerConfigJSON: %w", err)
		}
		result = append(result, parsedHosts...)
	}

	for _, h := range rc.Hosts {
		result = append(result, h.ToConfigHost())
	}

	result = append(result, rc.RegClientHosts...)

	return result, nil
}

type dockerConfigFile struct {
	Auths             map[string]dockerAuthConfig `json:"auths"`
	CredentialHelpers map[string]string           `json:"credHelpers"`
	CredentialsStore  string                      `json:"credsStore"`
}

type dockerAuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth,omitempty"`
	IdentityToken string `json:"identitytoken,omitempty"`
	RegistryToken string `json:"registrytoken,omitempty"`
}

func parseDockerConfigJSON(data []byte, allowCredentialHelpers bool) ([]config.Host, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "{}" {
		return nil, nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return nil, fmt.Errorf("invalid docker config JSON format: %w", err)
	}

	// Standard docker config.json contains "auths", "credHelpers", or "credsStore".
	_, hasAuths := raw["auths"]
	_, hasCredHelpers := raw["credHelpers"]
	_, hasCredsStore := raw["credsStore"]

	if hasAuths || hasCredHelpers || hasCredsStore {
		var dcf dockerConfigFile
		if err := json.Unmarshal([]byte(trimmed), &dcf); err != nil {
			return nil, fmt.Errorf("failed to parse docker config.json: %w", err)
		}

		if (len(dcf.CredentialHelpers) > 0 || dcf.CredentialsStore != "") && !allowCredentialHelpers {
			return nil, fmt.Errorf(
				"credential helper directives (credHelpers/credsStore) in DockerConfigJSON " +
					"are not allowed by default for security; set AllowCredentialHelpers to true to permit them",
			)
		}

		if len(dcf.Auths) == 0 && len(dcf.CredentialHelpers) == 0 && dcf.CredentialsStore == "" {
			// Present-but-empty auths/credHelpers maps without specific hosts or store
			return nil, nil
		}
		return hostsFromDockerConfig(dcf, allowCredentialHelpers)
	}

	// Try legacy .dockercfg format (top-level map of registry to auth config)
	var legacyAuths map[string]dockerAuthConfig
	if err := json.Unmarshal([]byte(trimmed), &legacyAuths); err == nil && len(legacyAuths) > 0 {
		return hostsFromDockerConfig(dockerConfigFile{Auths: legacyAuths}, false)
	}

	return nil, fmt.Errorf("invalid or unrecognized docker config JSON format")
}

func normalizeRegistryName(name string) string {
	cleanName := strings.TrimSpace(name)
	if cleanName == "index.docker.io" || cleanName == "https://index.docker.io/v1/" {
		return config.DockerRegistry
	}
	return cleanName
}

func formatHelper(helper string) string {
	if helper == "" {
		return ""
	}
	if strings.HasPrefix(helper, dockerHelperPrefix) {
		return helper
	}
	return dockerHelperPrefix + helper
}

func findHelper(dcf dockerConfigFile, name, cleanName string) string {
	if dcf.CredentialHelpers != nil {
		if h, ok := dcf.CredentialHelpers[name]; ok && h != "" {
			return formatHelper(h)
		}
		if h, ok := dcf.CredentialHelpers[cleanName]; ok && h != "" {
			return formatHelper(h)
		}
		if cleanName == config.DockerRegistry {
			if h, ok := dcf.CredentialHelpers["index.docker.io"]; ok && h != "" {
				return formatHelper(h)
			}
			if h, ok := dcf.CredentialHelpers["https://index.docker.io/v1/"]; ok && h != "" {
				return formatHelper(h)
			}
		}
	}
	if dcf.CredentialsStore != "" {
		return formatHelper(dcf.CredentialsStore)
	}
	return ""
}

func hostsFromDockerConfig(dcf dockerConfigFile, allowCredentialHelpers bool) ([]config.Host, error) {
	var hosts []config.Host
	seenHosts := make(map[string]bool)

	for name, auth := range dcf.Auths {
		cleanName := normalizeRegistryName(name)
		if cleanName == "" {
			continue
		}
		seenHosts[name] = true
		seenHosts[cleanName] = true

		user := auth.Username
		pass := auth.Password
		token := auth.IdentityToken
		if token == "" {
			token = auth.RegistryToken
		}

		if auth.Auth != "" && user == "" && pass == "" {
			decoded, err := base64.StdEncoding.DecodeString(auth.Auth)
			if err != nil {
				decoded, err = base64.RawStdEncoding.DecodeString(auth.Auth)
			}
			if err == nil {
				parts := strings.SplitN(string(decoded), ":", 2)
				user = parts[0]
				if len(parts) > 1 {
					pass = parts[1]
				}
			}
		}

		h := config.HostNewName(cleanName)
		h.User = user
		h.Pass = pass
		h.Token = token
		h.CredHelper = findHelper(dcf, name, cleanName)

		hosts = append(hosts, *h)
	}

	for name, helper := range dcf.CredentialHelpers {
		cleanName := normalizeRegistryName(name)
		if cleanName == "" {
			continue
		}
		if seenHosts[name] || seenHosts[cleanName] {
			continue
		}
		seenHosts[name] = true
		seenHosts[cleanName] = true

		h := config.HostNewName(cleanName)
		h.CredHelper = formatHelper(helper)
		hosts = append(hosts, *h)
	}

	if dcf.CredentialsStore != "" && allowCredentialHelpers {
		helper := formatHelper(dcf.CredentialsStore)
		cmd := exec.Command(helper, "list")
		cmd.Stdin = strings.NewReader("")
		if out, err := cmd.Output(); err == nil {
			var credList map[string]string
			if err := json.Unmarshal(out, &credList); err == nil {
				for reg, user := range credList {
					cleanName := normalizeRegistryName(reg)
					if cleanName == "" || seenHosts[reg] || seenHosts[cleanName] {
						continue
					}
					h := config.HostNewName(cleanName)
					if seenHosts[h.Name] {
						continue
					}
					seenHosts[reg] = true
					seenHosts[cleanName] = true
					seenHosts[h.Name] = true
					h.User = user
					h.CredHelper = helper
					hosts = append(hosts, *h)
				}
			}
		}
	}

	return hosts, nil
}
