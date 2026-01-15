// Copyright 2019 The kpt Authors
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

package cfgflags

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestUserAgentConfigFlags(t *testing.T) {
	testCases := []struct {
		name              string
		existingUserAgent string
		newUserAgent      string
		expectedUserAgent string
	}{
		{
			name:              "new useragent",
			existingUserAgent: "kubectl",
			newUserAgent:      "kpt",
			expectedUserAgent: "kpt",
		},
		{
			name:              "no new useragent",
			existingUserAgent: "kubectl",
			newUserAgent:      "",
			expectedUserAgent: "kubectl",
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			baseConfig := &rest.Config{
				UserAgent: tc.existingUserAgent,
			}

			cf := &UserAgentKubeConfigFlags{
				Delegate: &fakeRESTClientGetter{
					config: baseConfig,
				},
				UserAgent: tc.newUserAgent,
			}
			config, err := cf.ToRESTConfig()
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUserAgent, config.UserAgent)
		})
	}
}

type fakeRESTClientGetter struct {
	config *rest.Config
}

func (f *fakeRESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return f.config, nil
}

func (f *fakeRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return nil, nil
}

func (f *fakeRESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	return nil, nil
}

func (f *fakeRESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return nil
}
