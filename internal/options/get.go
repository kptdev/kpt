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

package options

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
)

// Get holds options for a list/get operation
type Get struct {
	*genericclioptions.ConfigFlags
	AllNamespaces bool
}

func (o *Get) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&o.AllNamespaces, "all-namespaces", "A", o.AllNamespaces,
		"If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
}

func (o *Get) ResourceBuilder() (*resource.Builder, error) {
	if *o.ConfigFlags.Namespace == "" {
		// Get the namespace from kubeconfig
		namespace, _, err := o.ConfigFlags.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return nil, fmt.Errorf("error getting namespace: %w", err)
		}
		o.ConfigFlags.Namespace = &namespace
	}

	b := resource.NewBuilder(o.ConfigFlags).
		NamespaceParam(*o.ConfigFlags.Namespace).AllNamespaces(o.AllNamespaces)
	return b, nil
}
