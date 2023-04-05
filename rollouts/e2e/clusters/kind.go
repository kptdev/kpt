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

package clusters

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type kindCluster struct {
	name           string
	labels         map[string]string
	kubeConfigPath string
}

type kindClusters struct {
	client.Client
	t         *testing.T
	cleanup   CleanupBehavior
	clusters  map[string]*kindCluster
	namespace string
}

var _ ClusterSetup = &kindClusters{}

func NewKindSetup(t *testing.T, c client.Client, cfgs ...Config) (ClusterSetup, error) {
	/*
		if len(cfg.Labels) > 0 {
			return nil, fmt.Errorf("labels are not supported with kind clusters")
		} */
	clusters := &kindClusters{
		t:         t,
		Client:    c,
		cleanup:   CleanupDelete,
		clusters:  make(map[string]*kindCluster),
		namespace: "kind-clusters",
	}
	for _, cfg := range cfgs {
		for i := 0; i < cfg.Count; i++ {
			clusterName := cfg.Prefix + fmt.Sprint(i)
			clusters.Add(clusterName, cfg.Labels)
		}
	}
	return clusters, nil
}

func (c *kindClusters) SetCleanupBehavior(cleanup CleanupBehavior) *kindClusters {
	c.cleanup = cleanup
	return c
}

// Add a cluster to the mix
func (c *kindClusters) Add(name string, labels map[string]string) error {
	c.clusters[name] = &kindCluster{
		name:   name,
		labels: labels,
	}
	return nil
}

// Wait for all clusters to become ready
func (c *kindClusters) PrepareAndWait(ctx context.Context, timeout time.Duration) error {
	c.t.Log("Ensure namespace exists for registering kind clusters")
	err := c.ensureNamespaceExists("kind-clusters")
	if err != nil {
		return err
	}
	c.t.Log("Verify kind is installed and kind command can be found")
	err = c.verifyKindIsInstalled()
	if err != nil {
		return err
	}
	c.t.Log("Verify test environment is clean")
	err = c.cleanTestEnvironment()
	if err != nil {
		return err
	}
	c.t.Log("Create kind cluster configuration file")
	clusterConfig, err := c.createClusterConfig()
	if err != nil {
		return err
	}
	defer os.Remove(clusterConfig)
	g, ctx := errgroup.WithContext(ctx)
	c.t.Log("Create kind clusters")
	for key := range c.clusters {
		clusterName := c.clusters[key].name
		labels := c.clusters[key].labels
		g.Go(func() error {
			return c.addCluster(ctx, clusterName, labels, clusterConfig)
		})
	}
	err = g.Wait()
	if err != nil {
		return err
	}
	return nil
}

// Cleanup deletes all clusters
func (c *kindClusters) Cleanup(ctx context.Context) error {
	if c.cleanup == CleanupDelete {
		err := c.cleanTestEnvironment()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *kindClusters) GetClusterRefs() []map[string]interface{} {
	crs := []map[string]interface{}{}
	for name := range c.clusters {
		cr := map[string]interface{}{"name": name}
		crs = append(crs, cr)
	}
	return crs
}

func (c *kindClusters) verifyKindIsInstalled() error {
	_, err := exec.LookPath("kind")
	if err != nil {
		return err
	}
	return nil
}

func (c *kindClusters) createClusterConfig() (string, error) {
	ipAddress, err := c.getHostIPAddress()
	if err != nil {
		return "", err
	}
	clusterConfigFile, err := os.CreateTemp("", "kind-cluster.yaml")
	if err != nil {
		return "", err
	}
	defer clusterConfigFile.Close()
	kindClusterConfig := `kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  # Allow connections to the API Sever with the CloudTop IP address
  apiServerAddress: "` + ipAddress + `"`
	bytes := []byte(kindClusterConfig)
	_, err = clusterConfigFile.Write(bytes)
	return clusterConfigFile.Name(), err
}

func (c *kindClusters) addCluster(ctx context.Context, name string, labels map[string]string, clusterConfig string) error {
	output, err := c.createCluster(ctx, name, clusterConfig)
	if err != nil {
		c.t.Log(string(output))
		return err
	}
	output, err = c.ensureConfigSyncIsInstalled(ctx, name)
	if err != nil {
		c.t.Log(string(output))
		return err
	}
	err = c.createConfigMapWithKubeConfig(name, labels)
	if err != nil {
		return err
	}
	return nil
}

func (c *kindClusters) createCluster(ctx context.Context, name, clusterConfig string) (string, error) {
	kubeConfigFile, err := os.CreateTemp("", "kubeconfig.yaml")
	if err != nil {
		return "", err
	}
	kubeConfig := kubeConfigFile.Name()
	// defer os.Remove(kubeConfig)
	// using the --kubeconfig flag as a hack to prevent kind from updating the kubeconfig context
	output, err := exec.Command("kind", "create", "cluster", "--name", name, "--config", clusterConfig, "--kubeconfig", kubeConfig).CombinedOutput()
	if err != nil {
		return string(output), err
	}
	c.clusters[name].kubeConfigPath = kubeConfig
	return string(output), nil
}

func (c *kindClusters) ensureConfigSyncIsInstalled(ctx context.Context, name string) (string, error) {
	kubeConfig := c.clusters[name].kubeConfigPath
	// defer os.Remove(kubeConfig)
	// using the --kubeconfig flag as a hack to prevent kind from updating the kubeconfig context
	c.t.Logf("installing configsync in the cluster %s", name)
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "https://github.com/GoogleContainerTools/kpt-config-sync/releases/download/v1.14.2/config-sync-manifest.yaml")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", kubeConfig))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	for {
		c.t.Logf("checking if configsync APIs are available in the cluster %s", name)
		cmd := exec.Command("kubectl", "get", "rootsyncs")
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", kubeConfig))
		_, err = cmd.CombinedOutput()
		if err == nil {
			return "", nil
		}
		time.Sleep(1 * time.Second)
	}
}

func (c *kindClusters) createConfigMapWithKubeConfig(name string, labels map[string]string) error {
	kubeConfigBytes, err := exec.Command("kind", "get", "kubeconfig", "--name", name).CombinedOutput()
	if err != nil {
		return err
	}
	kubeConfig := string(kubeConfigBytes)
	configMap := &unstructured.Unstructured{}
	configMap.SetGroupVersionKind(schema.GroupVersionKind{
		Version: "v1",
		Kind:    "ConfigMap",
	})
	configMap.SetName(name)
	configMap.SetNamespace(c.namespace)
	configMap.SetLabels(labels)
	unstructured.SetNestedField(configMap.Object, kubeConfig, "data", "kubeconfig.yaml")
	err = c.Client.Create(context.Background(), configMap, &client.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *kindClusters) deleteTestClusters() error {
	clusters, err := exec.Command("kind", "get", "clusters").Output()
	if err != nil {
		return err
	}
	clustersList := strings.Split(string(clusters), "\n")
	for _, kindClusterName := range clustersList {
		cluster, found := c.clusters[kindClusterName]
		if found {
			err := c.deleteCluster(kindClusterName)
			if err != nil {
				return err
			}
			err = os.Remove(cluster.kubeConfigPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *kindClusters) cleanTestEnvironment() error {
	// Delete Kind clusters
	err := c.deleteTestClusters()
	if err != nil {
		return err
	}
	// Delete KubeConfig config maps
	err = c.deleteTestConfigMaps()
	if err != nil {
		return err
	}
	return nil
}

func (c *kindClusters) deleteCluster(name string) error {
	err := exec.Command("kind", "delete", "cluster", "--name", name).Run()
	if err != nil {
		return err
	}
	return nil
}

func (c *kindClusters) deleteTestConfigMaps() error {
	for name := range c.clusters {
		clusterName := c.clusters[name].name
		configMap := &unstructured.Unstructured{}
		configMap.SetGroupVersionKind(schema.GroupVersionKind{
			Version: "v1",
			Kind:    "ConfigMap",
		})
		configMap.SetName(clusterName)
		configMap.SetNamespace(c.namespace)
		err := c.Client.Delete(context.Background(), configMap, &client.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *kindClusters) getHostIPAddress() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func (c *kindClusters) ensureNamespaceExists(namespace string) error {
	ns := corev1.Namespace{}
	nsKey := types.NamespacedName{Name: namespace}
	err := c.Client.Get(context.Background(), nsKey, &ns)
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}
	ns.Name = namespace
	err = c.Client.Create(context.Background(), &ns)
	if err != nil {
		return err
	}
	return nil
}
