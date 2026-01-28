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
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type UserAgentKubeConfigFlags struct {
	Delegate  genericclioptions.RESTClientGetter
	UserAgent string
}

func (u *UserAgentKubeConfigFlags) ToRESTConfig() (*rest.Config, error) {
	clientConfig, err := u.Delegate.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	if u.UserAgent != "" {
		clientConfig.UserAgent = u.UserAgent
	}
	return clientConfig, nil
}

func (u *UserAgentKubeConfigFlags) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return u.Delegate.ToDiscoveryClient()
}

func (u *UserAgentKubeConfigFlags) ToRESTMapper() (meta.RESTMapper, error) {
	return u.Delegate.ToRESTMapper()
}

func (u *UserAgentKubeConfigFlags) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return u.Delegate.ToRawKubeConfigLoader()
}
