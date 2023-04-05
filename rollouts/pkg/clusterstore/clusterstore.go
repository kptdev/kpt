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

package clusterstore

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gkeclusterapis "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/container/v1beta1"
	gkehubapis "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/gkehub/v1beta1"
	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
)

var (
	KCCClusterGVK         = gkeclusterapis.ContainerClusterGVK
	GKEFleetMembershipGVK = gkehubapis.GKEHubMembershipGVK
	KindClusterGVK        = schema.GroupVersionKind{
		Group:   "clusters.gitops.kpt.dev",
		Version: "v1",
		Kind:    "KindCluster",
	}
)

type ClusterStore struct {
	containerClusterStore *ContainerClusterStore
	gcpFleetClusterStore  *GCPFleetClusterStore
	kindClusterStore      *KindClusterStore
}

type Cluster struct {
	// Ref is the reference to the target cluster
	Ref    gitopsv1alpha1.ClusterRef
	Labels map[string]string
}

func NewClusterStore(client client.Client, config *rest.Config) (*ClusterStore, error) {
	containerClusterStore := &ContainerClusterStore{
		Config: config,
		Client: client,
	}
	if err := containerClusterStore.Init(); err != nil {
		return nil, err
	}

	clusterStore := &ClusterStore{
		containerClusterStore: containerClusterStore,
		gcpFleetClusterStore:  &GCPFleetClusterStore{},
		kindClusterStore:      &KindClusterStore{Client: client},
	}

	return clusterStore, nil
}

func (cs *ClusterStore) ListClusters(ctx context.Context, clusterDiscovery *gitopsv1alpha1.ClusterDiscovery, selector *metav1.LabelSelector) ([]Cluster, error) {
	clusterSourceType := clusterDiscovery.SourceType

	switch clusterSourceType {
	case gitopsv1alpha1.GCPFleet:
		return cs.gcpFleetClusterStore.ListClusters(ctx, clusterDiscovery.GCPFleet, selector)

	case gitopsv1alpha1.KCC:
		return cs.containerClusterStore.ListClusters(ctx, selector)

	case gitopsv1alpha1.KindCluster:
		return cs.kindClusterStore.ListClusters(ctx, selector)
	default:
		return nil, fmt.Errorf("%v cluster source not supported", clusterSourceType)
	}
}

func (cs *ClusterStore) GetRESTConfig(ctx context.Context, clusterRef *gitopsv1alpha1.ClusterRef) (*rest.Config, error) {
	// TODO (droot): Using kind property of the clusterRef for now but in the future
	// expand it to use the other properties (seems like an overkill for now).
	switch clusterKind := clusterRef.Kind; clusterKind {
	case GKEFleetMembershipGVK.Kind:
		return cs.gcpFleetClusterStore.GetRESTConfig(ctx, clusterRef.GetName())
	case KindClusterGVK.Kind:
		return cs.kindClusterStore.GetRESTConfig(ctx, clusterRef.GetName())
	case KCCClusterGVK.Kind:
		return cs.containerClusterStore.GetRESTConfig(ctx, clusterRef.GetName())
	default:
		return nil, fmt.Errorf("unknown cluster kind %s", clusterKind)
	}
}
