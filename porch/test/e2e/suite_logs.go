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

package e2e

import (
	"bytes"
	"context"
	"io"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *TestSuite) DumpLogsForDeployment(ctx context.Context, deploymentKey client.ObjectKey) {
	t.dumpLogsForDeployment(ctx, deploymentKey, t.Errorf)
}

func (c *TestSuite) hasOwner(child, parent runtime.Object) bool {
	childAccessor, err := meta.Accessor(child)
	if err != nil {
		c.Fatalf("could not get accessor for %T: %v", child, err)
	}
	parentAccessor, err := meta.Accessor(parent)
	if err != nil {
		c.Fatalf("could not get accessor for %T: %v", parent, err)
	}

	for _, ownerRef := range childAccessor.GetOwnerReferences() {
		// Kind is not always populated, UID should suffice (as it's globally unique)
		if ownerRef.UID != parentAccessor.GetUID() {
			continue
		}
		return true
	}

	return false
}

func (c *TestSuite) dumpLogsForDeployment(ctx context.Context, deploymentKey client.ObjectKey, eh ErrorHandler) {
	deployment, err := c.kubeClient.AppsV1().Deployments(deploymentKey.Namespace).Get(ctx, deploymentKey.Name, metav1.GetOptions{})
	if err != nil {
		eh("failed to get deployemnt %v: %v", deploymentKey, err)
	}

	replicaSets, err := c.kubeClient.AppsV1().ReplicaSets(deployment.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		eh("failed to list replicasets: %v", err)
	}

	for i := range replicaSets.Items {
		replicaSet := &replicaSets.Items[i]
		if !c.hasOwner(replicaSet, deployment) {
			continue
		}
		c.dumpLogsForReplicaSet(ctx, replicaSet, eh)
	}
}

func (c *TestSuite) dumpLogsForReplicaSet(ctx context.Context, replicaSet *appsv1.ReplicaSet, eh ErrorHandler) {
	pods, err := c.kubeClient.CoreV1().Pods(replicaSet.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		eh("failed to list pods: %v", err)
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		if !c.hasOwner(pod, replicaSet) {
			continue
		}
		c.dumpLogsForPod(ctx, pod, eh)
	}
}

func (c *TestSuite) dumpLogsForPod(ctx context.Context, pod *corev1.Pod, eh ErrorHandler) {
	for _, container := range pod.Spec.Containers {
		podKey := client.ObjectKey{
			Namespace: pod.Namespace,
			Name:      pod.Name,
		}
		c.dumpLogsForPodContainer(ctx, podKey, container.Name, eh)
	}
}

func (c *TestSuite) dumpLogsForPodContainer(ctx context.Context, podKey client.ObjectKey, containerName string, eh ErrorHandler) {
	req := c.kubeClient.CoreV1().Pods(podKey.Namespace).GetLogs(podKey.Name, &corev1.PodLogOptions{Container: containerName})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		eh("failed to open pod logs %v %s: %v", podKey, containerName, err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	if _, err = io.Copy(buf, podLogs); err != nil {
		eh("failed to copy pod logs %v %s: %v", podKey, containerName, err)
	}

	c.Logf("Logs from pod %v, container %s: %s", podKey, containerName, buf.String())
}
