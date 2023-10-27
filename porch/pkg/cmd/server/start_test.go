// Copyright 2023 The kpt Authors
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
	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/apiserver"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"testing"
	"time"
)

func TestAddFlags(t *testing.T) {
	versions := schema.GroupVersions{
		porchv1alpha1.SchemeGroupVersion,
	}
	o := PorchServerOptions{
		RecommendedOptions: genericoptions.NewRecommendedOptions(
			defaultEtcdPathPrefix,
			apiserver.Codecs.LegacyCodec(versions...),
		),
	}
	o.AddFlags(&pflag.FlagSet{})
	if o.RepoSyncFrequency < 30*time.Second {
		t.Fatalf("AddFlags(): repo-sync-frequency cannot be less that 30 seconds.")
	}
}
