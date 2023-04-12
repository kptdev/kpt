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
	"strings"
	"sync"

	"golang.org/x/oauth2/google"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	cloudresourcemanagerv1 "google.golang.org/api/cloudresourcemanager/v1"
	gkehubv1 "google.golang.org/api/gkehub/v1"
)

type GCPFleetClusterStore struct {
	projectId                       string
	membershipNameToConfigHostCache sync.Map
	projectIdToNumberCache          sync.Map
}

func (cs *GCPFleetClusterStore) ListClusters(ctx context.Context, configuration *gitopsv1alpha1.ClusterSourceGCPFleet, labelSelector *metav1.LabelSelector) ([]Cluster, error) {
	err := cs.validateGCPFleetConfiguration(configuration)
	if err != nil {
		return nil, fmt.Errorf("gcp fleet configuration error: %w", err)
	}

	clusters := []Cluster{}
	projectId := configuration.ProjectIds[0]

	// TODO: add support for listing meemberships for mulitple projects
	memberships, err := cs.listMemberships(ctx, projectId)
	if err != nil {
		return nil, fmt.Errorf("memberships failed: %w", err)
	}

	for _, membership := range memberships.Resources {
		cluster := cs.toCluster(membership)

		clusterLabelSet := labels.Set(cluster.Labels)
		shouldAdd := true

		if labelSelector != nil {
			selector, _ := metav1.LabelSelectorAsSelector(labelSelector)

			if !selector.Matches(clusterLabelSet) {
				shouldAdd = false
				continue
			}
		}

		if shouldAdd {
			clusters = append(clusters, cluster)
		}
	}

	return clusters, nil
}

func (cs *GCPFleetClusterStore) GetRESTConfig(ctx context.Context, name string) (*rest.Config, error) {
	accessToken, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("unable to get access token: %w", err)
	}

	token, err := accessToken.Token()

	host, err := cs.getRESTConfigHost(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("building rest config host failed: %w", err)
	}

	restConfig := &rest.Config{}
	restConfig.Host = host
	restConfig.BearerToken = token.AccessToken

	return restConfig, err
}

func (cs *GCPFleetClusterStore) getRESTConfigHost(ctx context.Context, name string) (string, error) {
	restConfigHost, found := cs.membershipNameToConfigHostCache.Load(name)

	if found {
		restConfigHost := restConfigHost.(string)
		return restConfigHost, nil
	}

	membership, err := cs.getMembership(ctx, name)
	if err != nil {
		return "", fmt.Errorf("unable to get membership: %w", err)
	}

	// name format: projects/:projectId/locations/global/memberships/:membershipName
	membershipName := strings.Split(membership.Name, "/")[5]
	projectId := strings.Split(membership.Name, "/")[1]

	isGKE := membership.Endpoint.GkeCluster != nil

	projectNumber, err := cs.getProjectNumber(ctx, projectId)
	if err != nil {
		return "", fmt.Errorf("unable to get project number: %w", err)
	}

	membershipUrl := "memberships"

	if isGKE {
		membershipUrl = "gkeMemberships"
	}

	host := fmt.Sprintf("https://connectgateway.googleapis.com/v1/projects/%d/locations/global/%s/%s", projectNumber, membershipUrl, membershipName)

	cs.membershipNameToConfigHostCache.Store(name, host)

	return host, nil
}

func (cs *GCPFleetClusterStore) getProjectNumber(ctx context.Context, projectId string) (int64, error) {
	projectNumberCache, found := cs.projectIdToNumberCache.Load(projectId)

	if found {
		projectNumber := projectNumberCache.(int64)
		return projectNumber, nil
	}

	crmClient, err := cloudresourcemanagerv1.NewService(ctx)
	if err != nil {
		return -1, fmt.Errorf("failed to create new cloudresourcemanager client: %w", err)
	}

	project, err := crmClient.Projects.Get(projectId).Context(ctx).Do()
	if err != nil {
		return -1, fmt.Errorf("error querying project %q: %w", projectId, err)
	}

	projectNumber := project.ProjectNumber

	cs.projectIdToNumberCache.Store(projectId, projectNumber)

	return projectNumber, nil
}

func (cs *GCPFleetClusterStore) validateGCPFleetConfiguration(configuration *gitopsv1alpha1.ClusterSourceGCPFleet) error {
	if configuration == nil {
		return fmt.Errorf("configuration is missing")
	}
	if len(configuration.ProjectIds) == 0 {
		return fmt.Errorf("at least one project id must be listed")
	}

	return nil
}

func (cs *GCPFleetClusterStore) getMembership(ctx context.Context, name string) (*gkehubv1.Membership, error) {
	hubClient, err := gkehubv1.NewService(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := hubClient.Projects.Locations.Memberships.Get(name).Do()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (cs *GCPFleetClusterStore) listMemberships(ctx context.Context, projectId string) (*gkehubv1.ListMembershipsResponse, error) {
	hubClient, err := gkehubv1.NewService(ctx)
	if err != nil {
		return nil, err
	}

	parent := fmt.Sprintf("projects/%s/locations/global", projectId)
	resp, err := hubClient.Projects.Locations.Memberships.List(parent).Do()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (cs *GCPFleetClusterStore) toCluster(membership *gkehubv1.Membership) Cluster {
	cluster := Cluster{
		Ref: gitopsv1alpha1.ClusterRef{
			APIVersion: GKEFleetMembershipGVK.GroupVersion().String(),
			Kind:       GKEFleetMembershipGVK.Kind,
			Name:       membership.Name,
		},
		Labels: membership.Labels,
	}

	return cluster
}
