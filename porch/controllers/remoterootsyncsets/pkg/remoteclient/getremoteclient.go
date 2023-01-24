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
	"os"

	container "cloud.google.com/go/container/apiv1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/googleurl"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	containerClusterKind       = "ContainerCluster"
	containerClusterApiVersion = "container.cnrm.cloud.google.com/v1beta1"

	configControllerKind       = "ConfigControllerInstance"
	configControllerApiVersion = "configcontroller.cnrm.cloud.google.com/v1beta1"
)

type RemoteClientGetter struct {
	client.Client

	workloadIdentity WorkloadIdentityHelper
}

// Init performs one-off initialization of the object.
func (r *RemoteClientGetter) Init(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()

	return r.workloadIdentity.Init(mgr.GetConfig())
}

// getCCRESTConfig builds a rest.Config for accessing the config controller cluster,
// this is a tmp workaround.
func (r *RemoteClientGetter) getCCRESTConfig(ctx context.Context, cluster *unstructured.Unstructured) (*rest.Config, error) {
	gkeResourceLink, _, err := unstructured.NestedString(cluster.Object, "status", "gkeResourceLink")
	if err != nil {
		return nil, fmt.Errorf("failed to get status.gkeResourceLink field: %w", err)
	}
	if gkeResourceLink == "" {
		return nil, fmt.Errorf("status.gkeResourceLink not set in object")
	}
	googleURL, err := googleurl.ParseUnversioned(gkeResourceLink)
	if err != nil {
		return nil, fmt.Errorf("error parsing gkeResourceLink %q: %w", gkeResourceLink, err)
	}
	projectID := googleURL.Project
	location := googleURL.Location
	clusterName := googleURL.Extra["clusters"]
	klog.Infof("cluster name is %s", clusterName)

	tokenSource, err := r.getConfigConnectorContextTokenSource(ctx, cluster.GetNamespace())
	if err != nil {
		return nil, err
	}

	gkeClient, err := container.NewClusterManagerClient(ctx, option.WithTokenSource(tokenSource), option.WithQuotaProject(projectID))
	if err != nil {
		return nil, fmt.Errorf("failed to create new cluster manager client: %w", err)
	}
	defer gkeClient.Close()

	clusterSelfLink := "projects/" + projectID + "/locations/" + location + "/clusters/" + clusterName
	klog.Infof("cluster path is %s", clusterSelfLink)
	req := &containerpb.GetClusterRequest{
		Name: clusterSelfLink,
	}

	resp, err := gkeClient.GetCluster(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get target cluster info: %w", err)
	}

	restConfig := &rest.Config{}
	caData, err := base64.StdEncoding.DecodeString(resp.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return nil, fmt.Errorf("error decoding ca certificate: %w", err)
	}
	restConfig.CAData = caData

	restConfig.Host = "https://" + resp.Endpoint
	klog.Infof("Host endpoint is %s", restConfig.Host)

	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("error getting token: %w", err)
	}

	restConfig.BearerToken = token.AccessToken
	return restConfig, nil
}

// getConfigConnectorContextTokenSource gets and returns the ConfigConnectorContext for the given namespace.
func (r *RemoteClientGetter) getConfigConnectorContextTokenSource(ctx context.Context, ns string) (oauth2.TokenSource, error) {
	if os.Getenv("USE_DEV_AUTH") != "" {
		klog.Warningf("using default authentication, intended for local development only")
		accessToken, err := GetDefaultAccessToken(ctx)
		if err != nil {
			return nil, err
		}
		return oauth2.StaticTokenSource(accessToken), nil
	}

	gvr := schema.GroupVersionResource{
		Group:    "core.cnrm.cloud.google.com",
		Version:  "v1beta1",
		Resource: "configconnectorcontexts",
	}

	id := types.NamespacedName{
		Namespace: ns,
		Name:      "configconnectorcontext.core.cnrm.cloud.google.com",
	}
	cr, err := r.workloadIdentity.dynamicClient.Resource(gvr).Namespace(id.Namespace).Get(ctx, id.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get configconnectorcontext %v: %w", id, err)
	}

	googleServiceAccount, _, err := unstructured.NestedString(cr.Object, "spec", "googleServiceAccount")
	if err != nil {
		return nil, fmt.Errorf("error reading spec.googleServiceAccount from ConfigConnectorContext in %q: %w", ns, err)
	}

	if googleServiceAccount == "" {
		return nil, fmt.Errorf("could not find spec.googleServiceAccount from ConfigConnectorContext in %q: %w", ns, err)
	}

	kubeServiceAccount := types.NamespacedName{
		Namespace: "cnrm-system",
		Name:      "cnrm-controller-manager-" + ns,
	}
	return r.workloadIdentity.GetGcloudAccessTokenSource(ctx, kubeServiceAccount, googleServiceAccount)
}

type completedReference struct {
	APIVersion string
	Kind       string
	Name       string
	Namespace  string
}

type Reference interface {
	GetAPIVersion() string
	GetKind() string
	GetName() string
	GetNamespace() string
}

func toCompletedReference(in Reference, defaultNamespace string) (completedReference, error) {
	ref := completedReference{
		Name:       in.GetName(),
		Namespace:  in.GetNamespace(),
		APIVersion: in.GetAPIVersion(),
		Kind:       in.GetKind(),
	}

	if ref.Namespace == "" {
		ref.Namespace = defaultNamespace
	}

	if ref.APIVersion == "" {
		switch ref.Kind {
		case containerClusterKind:
			ref.APIVersion = containerClusterApiVersion
		case configControllerKind:
			ref.APIVersion = configControllerApiVersion
		default:
			return completedReference{}, fmt.Errorf("clusterRef references unknown kind %q", ref.Kind)
		}
	}

	return ref, nil
}

type RemoteClient struct {
	restConfig *rest.Config
}

func (r *RemoteClient) DynamicClient() (dynamic.Interface, error) {
	dynamicClient, err := dynamic.NewForConfig(r.restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new dynamic client: %w", err)
	}
	return dynamicClient, nil
}

func (r *RemoteClient) RESTMapper() (meta.RESTMapper, error) {
	// TODO: Use a better discovery client
	discovery, err := discovery.NewDiscoveryClientForConfig(r.restConfig)
	if err != nil {
		return nil, fmt.Errorf("error building discovery client: %w", err)
	}

	cached := memory.NewMemCacheClient(discovery)

	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cached)
	return restMapper, nil
}

func (r *RemoteClientGetter) GetRemoteClient(ctx context.Context, clusterRef Reference, defaultNamespace string) (*RemoteClient, error) {
	ref, err := toCompletedReference(clusterRef, defaultNamespace)
	if err != nil {
		return nil, err
	}
	key := types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}

	u := &unstructured.Unstructured{}

	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse group version when building object: %w", err)
	}

	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    ref.Kind,
	})
	if err := r.Get(ctx, key, u); err != nil {
		return nil, fmt.Errorf("failed to get target cluster: %w", err)
	}

	var restConfig *rest.Config
	if ref.Kind == containerClusterKind {
		restConfig, err = r.getGKERESTConfig(ctx, u)
	} else if ref.Kind == configControllerKind {
		restConfig, err = r.getCCRESTConfig(ctx, u) //TODO: tmp workaround, update after ACP add new fields
	} else {
		return nil, fmt.Errorf("failed to find target cluster, cluster kind has to be ContainerCluster or ConfigControllerInstance")
	}
	if err != nil {
		return nil, err
	}

	remoteClient := &RemoteClient{
		restConfig: restConfig,
	}
	return remoteClient, nil
}

// getGKERESTConfig builds a rest.Config for accessing the specified cluster,
// without assuming that kubeconfig is correctly configured / mapped.
func (r *RemoteClientGetter) getGKERESTConfig(ctx context.Context, cluster *unstructured.Unstructured) (*rest.Config, error) {
	restConfig := &rest.Config{}

	clusterCaCertificate, exist, err := unstructured.NestedString(cluster.Object, "spec", "masterAuth", "clusterCaCertificate")
	if err != nil {
		return nil, fmt.Errorf("failed to get spec.masterAuth.clusterCaCertificate: %w", err)
	}
	if !exist {
		return nil, fmt.Errorf("spec.masterAuth.clusterCaCertificate field does not exist")
	}
	caData, err := base64.StdEncoding.DecodeString(clusterCaCertificate)
	if err != nil {
		return nil, fmt.Errorf("error decoding ca certificate: %w", err)
	}
	restConfig.CAData = caData

	endpoint, exist, err := unstructured.NestedString(cluster.Object, "status", "endpoint")
	if err != nil {
		return nil, fmt.Errorf("failed to get status.endpoint: %w", err)
	}
	if !exist {
		return nil, fmt.Errorf("status.endpoint field does not exist")
	}
	restConfig.Host = "https://" + endpoint
	klog.Infof("Host endpoint is %s", restConfig.Host)

	tokenSource, err := r.getConfigConnectorContextTokenSource(ctx, cluster.GetNamespace())
	if err != nil {
		return nil, fmt.Errorf("error building authentication token provider: %w", err)
	}
	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("error getting authentication token: %w", err)
	}
	restConfig.BearerToken = token.AccessToken

	return restConfig, nil
}

func GetDefaultAccessToken(ctx context.Context) (*oauth2.Token, error) {
	// Note: Not all tools support specifying the access token, so
	// the user still needs to log in with ADC.  e.g. terraform
	// https://github.com/hashicorp/terraform/issues/21680

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
