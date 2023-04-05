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

package apiserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KubeFunctionResolver resolves function names to full image paths
type KubeFunctionResolver struct {
	client             client.WithWatch
	defaultImagePrefix string
	// resolver  *FunctionResolver
	namespace string
}

// resolveToImagePorch converts the function short path to the full image url.
// If the function is Catalog function, it adds "gcr.io/kpt-fn/".e.g. set-namespace:v0.1 --> gcr.io/kpt-fn/set-namespace:v0.1
// If the function is porch function, it queries porch to get the function image by name and namespace.
// e.g. default:set-namespace:v0.1 --> us-west1-docker.pkg.dev/cpa-kit-dev/packages/set-namespace:v0.1
func (r *KubeFunctionResolver) resolveToImagePorch(ctx context.Context, image string) (string, error) {
	segments := strings.Split(image, ":")
	if len(segments) == 4 {
		// Porch function
		// TODO: Remove this legacy configuration
		functionName := strings.Join(segments[1:], ":")
		function, err := porch.FunctionGetter{}.Get(ctx, functionName, segments[0])
		if err != nil {
			return "", fmt.Errorf("failed to get image for function %q: %w", image, err)
		}
		return function.Spec.Image, nil
	}
	if !strings.Contains(image, "/") {
		var function v1alpha1.Function
		// TODO: Use fieldSelectors and better lookup
		name := "functions:" + image + ":latest"
		key := types.NamespacedName{
			Namespace: r.namespace,
			Name:      name,
		}
		// We query the apiserver for these types, even if we could query directly; this will then work with CRDs etc.
		// TODO: We need to think about priority-and-fairness with loopback queries
		if err := r.client.Get(ctx, key, &function); err != nil {
			if !apierrors.IsNotFound(err) {
				return "", fmt.Errorf("failed to get image for function %q: %w", image, err)
			}
		} else {
			return function.Spec.Image, nil
		}
		// TODO: Fallback to cluster-scoped?
		return r.defaultImagePrefix + image, nil
	}
	return image, nil
}
