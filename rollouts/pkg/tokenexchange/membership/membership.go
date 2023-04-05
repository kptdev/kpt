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

package membership

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var (
	gvr = schema.GroupVersionResource{
		Group:    "hub.gke.io",
		Version:  "v1",
		Resource: "memberships",
	}
)

// Membership is the object created by hub when a cluster is registered to hub.
type Membership struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MembershipSpec `json:"spec,omitempty"`
}

type MembershipSpec struct {
	Owner                MembershipOwner `json:"owner,omitempty"`
	WorkloadIdentityPool string          `json:"workload_identity_pool,omitempty"`
	IdentityProvider     string          `json:"identity_provider,omitempty"`
}

type MembershipOwner struct {
	ID string `json:"id,omitempty"`
}

// Get gets and returns the Membership named "membership" from the cluster.
func Get(ctx context.Context, client dynamic.Interface) (*Membership, error) {
	cr, err := client.Resource(gvr).Get(ctx, "membership", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// round-trip through JSON is a convenient way to get at structured content
	b, err := cr.MarshalJSON()
	if err != nil {
		return nil, err
	}
	membership := &Membership{}
	if err := json.Unmarshal(b, membership); err != nil {
		return nil, err
	}
	return membership, nil
}
