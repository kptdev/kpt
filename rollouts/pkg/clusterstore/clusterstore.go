// Copyright 2023 Google LLC
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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
)

type ClusterStore struct {
	containerClusterStore *ContainerClusterStore
	gcpFleetClusterStore  *GCPFleetClusterStore
}
type Cluster struct {
	Name   string
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

	default:
		return nil, fmt.Errorf("%v cluster source not supported", clusterSourceType)
	}
}

func (cs *ClusterStore) GetRESTConfig(ctx context.Context, name string) (*rest.Config, error) {
	switch {
	case strings.Contains(name, "memberships") || strings.Contains(name, "gkeMemberships"):
		return cs.gcpFleetClusterStore.GetRESTConfig(ctx, name)

	default:
		return cs.containerClusterStore.GetRESTConfig(ctx, name)
	}
}
