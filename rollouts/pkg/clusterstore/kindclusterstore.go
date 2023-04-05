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
	"encoding/base64"
	"fmt"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kindClustersNamespace = "kind-clusters"
)

type KindClusterStore struct {
	// Client points to the config
	client.Client
}

func (cs *KindClusterStore) ListClusters(ctx context.Context, selector *metav1.LabelSelector) ([]Cluster, error) {
	kindClusters := &unstructured.UnstructuredList{}
	kindClusters.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMapList",
	})

	var opts []client.ListOption
	if selector != nil {
		selector, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: selector})
	}

	// TODO(droot): Make it configurable when needed.
	namespace := kindClustersNamespace

	opts = append(opts, client.InNamespace(namespace))

	if err := cs.List(ctx, kindClusters, opts...); err != nil {
		return nil, err
	}

	clusters := []Cluster{}

	for _, kindCluster := range kindClusters.Items {
		kc := kindCluster
		cluster := cs.toCluster(&kc)
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

func (cs *KindClusterStore) toCluster(kindCluster *unstructured.Unstructured) Cluster {
	cluster := Cluster{
		Ref: gitopsv1alpha1.ClusterRef{
			APIVersion: KindClusterGVK.GroupVersion().String(),
			Kind:       KindClusterGVK.Kind,
			Name:       kindCluster.GetName(),
		},
		Labels: kindCluster.GetLabels(),
	}
	return cluster
}

func (cs *KindClusterStore) GetRESTConfig(ctx context.Context, name string) (*rest.Config, error) {
	kindCluster := unstructured.Unstructured{}
	kindCluster.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	})

	clusterKey := client.ObjectKey{
		Namespace: kindClustersNamespace,
		Name:      name,
	}
	if err := cs.Get(ctx, clusterKey, &kindCluster); err != nil {
		return nil, err
	}

	kubeConfig, err := kubeConfigFromConfigMap(kindCluster)
	if err != nil {
		return nil, err
	}
	restConfig, err := kubeConfigToRESTConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	return restConfig, err
}

func kubeConfigFromConfigMap(configMap unstructured.Unstructured) (*kubeConfig, error) {
	configMapData, exists, err := unstructured.NestedStringMap(configMap.Object, "data")
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("data not found")
	}
	kubeConfigString, exists := configMapData["kubeconfig.yaml"]
	if !exists {
		return nil, fmt.Errorf("kubeconfig.yaml not found")
	}
	kubeConfig := &kubeConfig{}
	yaml.Unmarshal([]byte(kubeConfigString), &kubeConfig)
	return kubeConfig, err
}

func kubeConfigToRESTConfig(config *kubeConfig) (*rest.Config, error) {
	if len(config.Clusters) != 1 {
		return nil, fmt.Errorf("kubeconfig contain one and only one cluster")
	}
	if len(config.Users) != 1 {
		return nil, fmt.Errorf("kubeconfig must contain one and only one user")
	}
	cluster := config.Clusters[0].Cluster
	user := config.Users[0].User
	if cluster.CAData == "" || user.CertificateData == "" || user.KeyData == "" {
		return nil, fmt.Errorf("kubeconfig does not contain required certificate data")
	}
	caData, err := base64.StdEncoding.DecodeString(cluster.CAData)
	if err != nil {
		return nil, err
	}
	certData, err := base64.StdEncoding.DecodeString(user.CertificateData)
	if err != nil {
		return nil, err
	}
	keyData, err := base64.StdEncoding.DecodeString(user.KeyData)
	if err != nil {
		return nil, err
	}
	restConfig := rest.Config{}
	restConfig.Host = cluster.Server
	restConfig.CAData = caData
	restConfig.CertData = certData
	restConfig.KeyData = keyData
	return &restConfig, nil
}

// internal datastructures to help with unmarshalling of the kubeConfig

type kubeConfig struct {
	Clusters []configCluster `yaml:"clusters"`
	Users    []configUser    `yaml:"users"`
}
type configCluster struct {
	Cluster clusterConfig `yaml:"cluster"`
}
type clusterConfig struct {
	Server string `yaml:"server"`
	CAData string `yaml:"certificate-authority-data"`
}
type configUser struct {
	User userConfig `yaml:"user"`
}
type userConfig struct {
	KeyData         string `yaml:"client-key-data"`
	CertificateData string `yaml:"client-certificate-data"`
}
