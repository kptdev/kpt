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

package porch

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdatePackageRevisionApproval(ctx context.Context, client rest.Interface, key client.ObjectKey, new v1alpha1.PackageRevisionLifecycle) error {
	scheme := runtime.NewScheme()
	if err := v1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}

	codec := runtime.NewParameterCodec(scheme)
	var pr v1alpha1.PackageRevision
	if err := client.Get().
		Namespace(key.Namespace).
		Resource("packagerevisions").
		Name(key.Name).
		VersionedParams(&metav1.GetOptions{}, codec).
		Do(ctx).
		Into(&pr); err != nil {
		return err
	}

	switch lifecycle := pr.Spec.Lifecycle; lifecycle {
	case v1alpha1.PackageRevisionLifecycleProposed, v1alpha1.PackageRevisionLifecycleDeletionProposed:
		// ok
	case new:
		// already correct value
		return nil
	default:
		return fmt.Errorf("cannot change approval from %s to %s", lifecycle, new)
	}

	// Approve - change the package revision kind to "final".
	pr.Spec.Lifecycle = new

	opts := metav1.UpdateOptions{}
	result := &v1alpha1.PackageRevision{}
	return client.Put().
		Namespace(pr.Namespace).
		Resource("packagerevisions").
		Name(pr.Name).
		SubResource("approval").
		VersionedParams(&opts, codec).
		Body(&pr).
		Do(ctx).
		Into(result)
}
