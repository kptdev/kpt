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

package main

import (
	"context"
	goflag "flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func main() {
	err := run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

type RootOptions struct {
}

func run(ctx context.Context) error {
	var opt RootOptions

	rootCmd := &cobra.Command{
		Use:           "update-package-resources",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	fs := goflag.NewFlagSet("", goflag.PanicOnError)
	klog.InitFlags(fs)
	rootCmd.PersistentFlags().AddGoFlagSet(fs)

	AddUpdatePackageResourcesCommand(rootCmd, opt)

	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}

func GetRESTConfig() (*rest.Config, error) {
	// First try an in-cluster config
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig configuration
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{}).ClientConfig()
}
