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

package apiserver

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/install"
	"github.com/GoogleContainerTools/kpt/porch/apiserver/pkg/registry/porch"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/engine"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/cache"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme = runtime.NewScheme()
	// Codecs provides methods for retrieving codecs and serializers for specific
	// versions and content types.
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	install.Install(Scheme)

	// we need to add the options to empty v1
	// TODO fix the server code to avoid this
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

// ExtraConfig holds custom apiserver config
type ExtraConfig struct {
	CoreAPIKubeconfigPath string
	CacheDirectory        string
	FunctionRunnerAddress string
}

// Config defines the config for the apiserver
type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

// PorchServer contains state for a Kubernetes cluster master/api server.
type PorchServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
	coreClient       client.WithWatch
	cache            *cache.Cache
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *Config) Complete() CompletedConfig {
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}

	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&c}
}

func (c completedConfig) getCoreClient() (client.WithWatch, error) {
	var restConfig *rest.Config

	kubeconfig := c.ExtraConfig.CoreAPIKubeconfigPath
	if kubeconfig == "" {
		icc, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load in-cluster config (specify --kubeconfig if not running in-cluster): %w", err)
		}
		restConfig = icc
	} else {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

		cc, err := loader.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load config %q: %w", kubeconfig, err)
		}
		restConfig = cc
	}

	// set high qps/burst limits since this will effectively limit API server responsiveness
	restConfig.QPS = 200
	restConfig.Burst = 400

	scheme := runtime.NewScheme()
	if err := configapi.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("error building scheme: %w", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("error building scheme: %w", err)
	}

	coreClient, err := client.NewWithWatch(restConfig, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("error building client for core apiserver: %w", err)
	}

	return coreClient, nil
}

// New returns a new instance of PorchServer from the given config.
func (c completedConfig) New() (*PorchServer, error) {
	genericServer, err := c.GenericConfig.New("sample-apiserver", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	coreClient, err := c.getCoreClient()
	if err != nil {
		return nil, fmt.Errorf("failed to build client for core apiserver: %w", err)
	}

	credentialResolver := porch.NewCredentialResolver(coreClient)

	cache := cache.NewCache(c.ExtraConfig.CacheDirectory, credentialResolver)
	cad, err := engine.NewCaDEngine(
		engine.WithCache(cache),
		engine.WithGRPCFunctionRuntime(c.ExtraConfig.FunctionRunnerAddress),
		engine.WithCredentialResolver(credentialResolver),
	)

	porchGroup, err := porch.NewRESTStorage(Scheme, Codecs, cad, coreClient)
	if err != nil {
		return nil, err
	}

	s := &PorchServer{
		GenericAPIServer: genericServer,
		coreClient:       coreClient,
		cache:            cache,
	}

	// Install the groups.
	if err := s.GenericAPIServer.InstallAPIGroups(&porchGroup); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *PorchServer) Run(stopCh <-chan struct{}) error {
	porch.RunBackground(s.coreClient, s.cache, stopCh)
	return s.GenericAPIServer.PrepareRun().Run(stopCh)
}
