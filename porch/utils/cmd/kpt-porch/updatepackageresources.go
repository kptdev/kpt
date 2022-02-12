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
	"fmt"
	"io/ioutil"
	"os"

	"github.com/GoogleContainerTools/kpt/porch/apiserver/pkg/generated/clientset/versioned"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

type UpdatePackageResourcesOptions struct {
	Source string

	Namespace string
	Name      string
}

func AddUpdatePackageResourcesCommand(parent *cobra.Command, parentOptions RootOptions) *cobra.Command {
	var opt UpdatePackageResourcesOptions

	cmd := &cobra.Command{
		Use: "update-package-resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			opt.Name = args[0]
			return RunUpdatePackagesResources(cmd.Context(), opt)
		},
		Args: cobra.ExactArgs(1),
	}

	cmd.Flags().StringVarP(&opt.Source, "filename", "f", "", "directory to read from (or - for stdin)")
	cmd.MarkFlagRequired("filename")

	cmd.Flags().StringVarP(&opt.Namespace, "namespace", "n", "", "namespace to use")

	parent.AddCommand(cmd)

	return cmd
}

func RunUpdatePackagesResources(ctx context.Context, opt UpdatePackageResourcesOptions) error {
	restConfig, err := GetRESTConfig()
	if err != nil {
		return fmt.Errorf("error getting kubernetes configuration: %w", err)
	}

	clientset, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}
	porch := clientset.PorchV1alpha1()
	id := types.NamespacedName{
		Namespace: opt.Namespace,
		Name:      opt.Name,
	}

	packageRevision, err := porch.PackageRevisionResources(id.Namespace).Get(ctx, id.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get package resources %v: %w", id, err)
	}

	updatedResources := make(map[string]string)

	if opt.Source == "-" {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("error reading from stdin: %w", err)
		}
		updatedResources["manifest.yaml"] = string(b)
	} else {
		return fmt.Errorf("only stdin source is currently implemented")
	}

	packageRevision.Spec.Resources = updatedResources
	// TODO: Use server-side-apply (also ... who implements SSA in an aggregated apiserver?)
	updated, err := porch.PackageRevisionResources(id.Namespace).Update(ctx, packageRevision, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update package resources %v: %w", id, err)
	}
	klog.Infof("updated %v to %s", id, updated.GetResourceVersion())

	return nil
}
