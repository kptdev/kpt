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
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/util/porch"
)

// resolveToImagePorch converts the function short path to the full image url.
// If the function is Catalog function, it adds "gcr.io/kpt-fn/".e.g. set-namespace:v0.1 --> gcr.io/kpt-fn/set-namespace:v0.1
// If the function is porch function, it queries porch to get the function image by name and namespace.
// e.g. default:set-namespace:v0.1 --> us-west1-docker.pkg.dev/cpa-kit-dev/packages/set-namespace:v0.1
func resolveToImagePorch(ctx context.Context, image string) (string, error) {
	segments := strings.Split(image, ":")
	if len(segments) == 4 {
		// Porch function
		functionName := strings.Join(segments[1:], ":")
		function, err := porch.FunctionGetter{}.Get(ctx, functionName, segments[0])
		if err != nil {
			return "", fmt.Errorf("failed to resolve image: %w", err)
		}
		return function.Spec.Image, nil
	}
	if !strings.Contains(image, "/") {
		return fmt.Sprintf("gcr.io/kpt-fn/%s", image), nil
	}
	return image, nil
}
