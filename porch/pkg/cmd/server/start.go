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

package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	clientset "github.com/GoogleContainerTools/kpt/porch/api/generated/clientset/versioned"
	informers "github.com/GoogleContainerTools/kpt/porch/api/generated/informers/externalversions"
	sampleopenapi "github.com/GoogleContainerTools/kpt/porch/api/generated/openapi"
	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/apiserver"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/features"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/klog/v2"
	netutils "k8s.io/utils/net"
)

const defaultEtcdPathPrefix = "/registry/porch.kpt.dev"

// PorchServerOptions contains state for master/api server
type PorchServerOptions struct {
	RecommendedOptions       *genericoptions.RecommendedOptions
	LocalStandaloneDebugging bool // Enables local standalone running/debugging of the apiserver.
	CacheDirectory           string
	CoreAPIKubeconfigPath    string
	FunctionRunnerAddress    string
	DefaultImagePrefix       string
	RepoSyncFrequency        time.Duration

	SharedInformerFactory informers.SharedInformerFactory
	StdOut                io.Writer
	StdErr                io.Writer
}

// NewPorchServerOptions returns a new PorchServerOptions
func NewPorchServerOptions(out, errOut io.Writer) *PorchServerOptions {
	//
	// GroupVersions served by this server
	//
	versions := schema.GroupVersions{
		porchv1alpha1.SchemeGroupVersion,
	}

	o := &PorchServerOptions{
		RecommendedOptions: genericoptions.NewRecommendedOptions(
			defaultEtcdPathPrefix,
			apiserver.Codecs.LegacyCodec(versions...),
		),

		StdOut: out,
		StdErr: errOut,
	}
	o.RecommendedOptions.Etcd.StorageConfig.EncodeVersioner = versions
	o.RecommendedOptions.Etcd = nil
	return o
}

// NewCommandStartPorchServer provides a CLI handler for 'start master' command
// with a default PorchServerOptions.
func NewCommandStartPorchServer(ctx context.Context, defaults *PorchServerOptions) *cobra.Command {
	o := *defaults
	cmd := &cobra.Command{
		Short: "Launch a porch API server",
		Long:  "Launch a porch API server",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.RunPorchServer(ctx); err != nil {
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()
	o.AddFlags(flags)

	return cmd
}

// Validate validates PorchServerOptions
func (o PorchServerOptions) Validate(args []string) error {
	errors := []error{}
	errors = append(errors, o.RecommendedOptions.Validate()...)
	return utilerrors.NewAggregate(errors)
}

// Complete fills in fields required to have valid data
func (o *PorchServerOptions) Complete() error {
	o.CoreAPIKubeconfigPath = o.RecommendedOptions.CoreAPI.CoreAPIKubeconfigPath

	if o.LocalStandaloneDebugging {
		if os.Getenv("KUBERNETES_SERVICE_HOST") != "" || os.Getenv("KUBERNETES_SERVICE_PORT") != "" {
			klog.Fatalf("--standalone-debug-mode must not be used when running in k8s")
		} else {
			o.RecommendedOptions.Authorization = nil
			o.RecommendedOptions.CoreAPI = nil
			o.RecommendedOptions.Admission = nil
			o.RecommendedOptions.Authentication.RemoteKubeConfigFileOptional = true
		}
	}

	if o.CacheDirectory == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			cache = os.TempDir()
			klog.Warningf("Cannot find user cache directory, using temporary directory %q", cache)
		}
		o.CacheDirectory = cache + "/porch"
	}

	// if !o.LocalStandaloneDebugging {
	// 	TODO: register admission plugins here ...
	// 	add admission plugins to the RecommendedPluginOrder here ...
	// }

	return nil
}

// Config returns config for the api server given PorchServerOptions
func (o *PorchServerOptions) Config() (*apiserver.Config, error) {
	// TODO have a "real" external address
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %w", err)
	}

	if o.RecommendedOptions.Etcd != nil {
		o.RecommendedOptions.Etcd.StorageConfig.Paging = utilfeature.DefaultFeatureGate.Enabled(features.APIListChunking)
	}

	o.RecommendedOptions.ExtraAdmissionInitializers = func(c *genericapiserver.RecommendedConfig) ([]admission.PluginInitializer, error) {
		client, err := clientset.NewForConfig(c.LoopbackClientConfig)
		if err != nil {
			return nil, err
		}
		informerFactory := informers.NewSharedInformerFactory(client, c.LoopbackClientConfig.Timeout)
		o.SharedInformerFactory = informerFactory
		return []admission.PluginInitializer{}, nil
	}

	serverConfig := genericapiserver.NewRecommendedConfig(apiserver.Codecs)

	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(sampleopenapi.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(apiserver.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "Porch"
	serverConfig.OpenAPIConfig.Info.Version = "0.1"

	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	config := &apiserver.Config{
		GenericConfig: serverConfig,
		ExtraConfig: apiserver.ExtraConfig{
			CoreAPIKubeconfigPath: o.CoreAPIKubeconfigPath,
			CacheDirectory:        o.CacheDirectory,
			RepoSyncFrequency:     o.RepoSyncFrequency,
			FunctionRunnerAddress: o.FunctionRunnerAddress,
			DefaultImagePrefix:    o.DefaultImagePrefix,
		},
	}
	return config, nil
}

// RunPorchServer starts a new PorchServer given PorchServerOptions
func (o PorchServerOptions) RunPorchServer(ctx context.Context) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}

	if config.GenericConfig.SharedInformerFactory != nil {
		server.GenericAPIServer.AddPostStartHookOrDie("start-sample-server-informers", func(context genericapiserver.PostStartHookContext) error {
			config.GenericConfig.SharedInformerFactory.Start(context.StopCh)
			o.SharedInformerFactory.Start(context.StopCh)
			return nil
		})
	}

	return server.Run(ctx)
}

func (o *PorchServerOptions) AddFlags(fs *pflag.FlagSet) {
	// Add base flags
	o.RecommendedOptions.AddFlags(fs)
	utilfeature.DefaultMutableFeatureGate.AddFlag(fs)

	// Add additional flags.

	if os.Getenv("KUBERNETES_SERVICE_HOST") == "" && os.Getenv("KUBERNETES_SERVICE_PORT") == "" {
		// Add this flag only when not running in k8s cluster.
		fs.BoolVar(&o.LocalStandaloneDebugging, "standalone-debug-mode", false,
			"Under the local-debug mode the apiserver will allow all access to its resources without "+
				"authorizing the requests, this flag is only intended for debugging in your workstation.")
	}

	fs.StringVar(&o.FunctionRunnerAddress, "function-runner", "", "Address of the function runner gRPC service.")
	fs.StringVar(&o.DefaultImagePrefix, "default-image-prefix", "gcr.io/kpt-fn/", "Default prefix for unqualified function names")
	fs.StringVar(&o.CacheDirectory, "cache-directory", "", "Directory where Porch server stores repository and package caches.")
	fs.DurationVar(&o.RepoSyncFrequency, "repo-sync-frequency", 60*time.Second, "Frequency in seconds at which registered repositories will be synced.")
}
