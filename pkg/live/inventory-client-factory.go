// Copyright 2022,2026 The kpt Authors
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

package live

import (
	"context"

	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/inventory"
)

// ClusterClientFactory is a factory that creates instances of ClusterClient
// inventory client.
//
// Ctx, if set, is plumbed into the InventoryResourceGroup wrapper so
// Apply / ApplyWithPrune honor caller cancellation (Ctrl-C, timeouts).
// The upstream inventory.ClientFactory interface's NewClient signature
// does not accept a context, so we carry one on the factory instead;
// construct via NewClusterClientFactoryWithContext when you have one.
type ClusterClientFactory struct {
	StatusPolicy inventory.StatusPolicy
	Ctx          context.Context
}

// NewClusterClientFactory returns a ClusterClientFactory that will build
// inventory clients with no context propagation (cluster API calls use
// context.Background()). Prefer NewClusterClientFactoryWithContext.
func NewClusterClientFactory() *ClusterClientFactory {
	return &ClusterClientFactory{StatusPolicy: inventory.StatusPolicyNone}
}

// NewClusterClientFactoryWithContext returns a ClusterClientFactory that
// threads ctx into every inventory client it produces.
//
// A nil ctx is normalized to context.Background() so the docstring's
// promise ("threads ctx into every inventory client") holds for every
// input: there is no hidden code path that silently drops propagation.
func NewClusterClientFactoryWithContext(ctx context.Context) *ClusterClientFactory {
	if ctx == nil {
		ctx = context.Background()
	}
	return &ClusterClientFactory{StatusPolicy: inventory.StatusPolicyNone, Ctx: ctx}
}

func (ccf *ClusterClientFactory) NewClient(factory cmdutil.Factory) (inventory.Client, error) {
	// Defense in depth: normalize a nil Ctx here too. This covers the
	// case where a caller constructed ClusterClientFactory as a struct
	// literal (e.g. &ClusterClientFactory{StatusPolicy: ...}) and left
	// Ctx unset — see NewClusterClientFactory, which does exactly that.
	ctx := ccf.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return inventory.NewClient(factory, WrapInventoryObjWithContext(ctx), InvToUnstructuredFunc, ccf.StatusPolicy, ResourceGroupGVK)
}
