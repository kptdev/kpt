/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ client.Client = &fakeRolloutsClient{}

type fakeRolloutsClient struct {
	client.Client

	remotesyncs []gitopsv1alpha1.RemoteSync
	actions     []string
}

func (fc *fakeRolloutsClient) Create(ctx context.Context, obj client.Object, _ ...client.CreateOption) error {
	fc.actions = append(fc.actions, fmt.Sprintf("creating object named %q", obj.GetName()))

	if err := fc.Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj); err == nil {
		// object already exists
		return fmt.Errorf("object %q already exists", obj.GetName())
	}
	fc.remotesyncs = append(fc.remotesyncs, *obj.(*gitopsv1alpha1.RemoteSync))

	return nil
}

func (fc *fakeRolloutsClient) Delete(_ context.Context, obj client.Object, _ ...client.DeleteOption) error {
	fc.actions = append(fc.actions, fmt.Sprintf("deleting object named %q", obj.GetName()))

	var newRemoteSyncs []gitopsv1alpha1.RemoteSync
	for _, rs := range fc.remotesyncs {
		if obj.GetName() == rs.GetName() && obj.GetNamespace() == rs.GetNamespace() {
			continue
		}
		newRemoteSyncs = append(newRemoteSyncs, rs)
	}
	fc.remotesyncs = newRemoteSyncs

	return nil
}

func (fc *fakeRolloutsClient) Update(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
	fc.actions = append(fc.actions, fmt.Sprintf("updating object named %q", obj.GetName()))

	for i, rs := range fc.remotesyncs {
		if obj.GetName() == rs.GetName() && obj.GetNamespace() == rs.GetNamespace() {
			continue
		}
		fc.remotesyncs[i] = *obj.(*gitopsv1alpha1.RemoteSync)
	}

	return nil
}

func (fc *fakeRolloutsClient) List(_ context.Context, obj client.ObjectList, _ ...client.ListOption) error {
	fc.actions = append(fc.actions, fmt.Sprintf("listing objects"))

	*obj.(*gitopsv1alpha1.RemoteSyncList) = gitopsv1alpha1.RemoteSyncList{Items: fc.remotesyncs}

	return nil
}

func (fc *fakeRolloutsClient) Get(_ context.Context, namespacedname types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
	fc.actions = append(fc.actions, fmt.Sprintf("getting object of name %q", namespacedname.Name))

	for _, rs := range fc.remotesyncs {
		if rs.GetName() == namespacedname.Name && namespacedname.Namespace == rs.GetNamespace() {
			*obj.(*gitopsv1alpha1.RemoteSync) = rs
			return nil
		}
	}

	return &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
}

func (fc *fakeRolloutsClient) setSyncStatus(name string, namespace string, syncStatus string) error {
	for i, rs := range fc.remotesyncs {
		if rs.GetName() == name && namespace == rs.GetNamespace() {
			rs.Status.SyncStatus = syncStatus
			fc.remotesyncs[i] = rs
			return nil
		}
	}
	return &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
}
