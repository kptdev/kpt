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

package remoteclient

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"

	container "cloud.google.com/go/container/apiv1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsync/api/v1alpha1"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetCCRESTConfig builds a rest.Config for accessing the config controller cluster,
// this is a tmp workaround.
func GetCCRESTConfig(ctx context.Context, cluster *unstructured.Unstructured) (*rest.Config, error) {
	gkeResourceLink, exist, err := unstructured.NestedString(cluster.Object, "status", "gkeResourceLink")
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}
	if !exist {
		return nil, fmt.Errorf("failed to find gkeResourceLink field")
	}
	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create new cluster manager client: %w", err)
	}
	defer c.Close()
	nameMatchPattern := regexp.MustCompile(`projects/.*`)
	clusterName := nameMatchPattern.FindString(gkeResourceLink)
	klog.Infof("cluster name is %s", clusterName)
	req := &containerpb.GetClusterRequest{
		Name: clusterName,
	}
	resp, err := c.GetCluster(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster info: %w", err)
	}
	restConfig := &rest.Config{}
	caData, err := base64.StdEncoding.DecodeString(resp.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return nil, fmt.Errorf("error decoding ca certificate: %w", err)
	}
	restConfig.CAData = caData

	restConfig.Host = "https://" + resp.Endpoint
	klog.Infof("Host endpoint is %s", restConfig.Host)
	accessToken, err := GetGcloudAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	restConfig.BearerToken = accessToken.AccessToken
	return restConfig, nil
}

func GetRemoteClient(ctx context.Context, c client.Client, ref *api.ClusterRef, ns string) (*rest.Config, error) {
	key := types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}
	if key.Namespace == "" {
		key.Namespace = ns
	}
	u := &unstructured.Unstructured{}
	var config *rest.Config
	gv, err := schema.ParseGroupVersion(ref.ApiVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse group version when building object: %w", err)
	}

	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    ref.Kind,
	})
	if err := c.Get(ctx, key, u); err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}
	if ref.Kind == "ContainerCluster" {
		config, err = GetGKERESTConfig(ctx, u)
	} else if ref.Kind == "ConfigControllerInstance" {
		config, err = GetCCRESTConfig(ctx, u) //TODO: tmp workaround, update after ACP add new fields
	} else {
		return nil, fmt.Errorf("failed to find target cluster, cluster kind has to be ContainerCluster or ConfigControllerInstance")
	}
	if err != nil {
		return nil, err
	}
	return config, nil
}

// GetGKERESTConfig builds a rest.Config for accessing the specified cluster,
// without assuming that kubeconfig is correctly configured / mapped.
func GetGKERESTConfig(ctx context.Context, cluster *unstructured.Unstructured) (*rest.Config, error) {
	restConfig := &rest.Config{}
	clusterCaCertificate, exist, err := unstructured.NestedString(cluster.Object, "spec", "masterAuth", "clusterCaCertificate")
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}
	if !exist {
		return nil, fmt.Errorf("clusterCaCertificate field does not exist")
	}
	caData, err := base64.StdEncoding.DecodeString(clusterCaCertificate)
	if err != nil {
		return nil, fmt.Errorf("error decoding ca certificate: %w", err)
	}
	restConfig.CAData = caData
	endpoint, exist, err := unstructured.NestedString(cluster.Object, "status", "endpoint")
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}
	if !exist {
		return nil, fmt.Errorf("endpoint field does not exist")
	}
	restConfig.Host = "https://" + endpoint
	klog.Infof("Host endpoint is %s", restConfig.Host)
	accessToken, err := GetGcloudAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	restConfig.BearerToken = accessToken.AccessToken
	return restConfig, nil
}

func GetGcloudAccessToken(ctx context.Context) (*oauth2.Token, error) {
	accessToken, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("unable to get default access-token from gcloud: %w", err)
	}
	token, err := accessToken.Token()
	if err != nil {
		return nil, fmt.Errorf("unable to get token from token source: %w", err)
	}

	return &oauth2.Token{
		AccessToken: token.AccessToken,
	}, nil
}
