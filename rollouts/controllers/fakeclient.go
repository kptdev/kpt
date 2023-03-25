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

var _ client.Client = &fakeRemoteSyncClient{}

type fakeRemoteSyncClient struct {
	client.Client

	remotesyncs map[types.NamespacedName]gitopsv1alpha1.RemoteSync
	actions     []string
}

func newFakeRemoteSyncClient() *fakeRemoteSyncClient {
	return &fakeRemoteSyncClient{
		remotesyncs: make(map[types.NamespacedName]gitopsv1alpha1.RemoteSync),
	}
}

func (fc *fakeRemoteSyncClient) Create(ctx context.Context, obj client.Object, _ ...client.CreateOption) error {
	fc.actions = append(fc.actions, fmt.Sprintf("creating object named %q", obj.GetName()))

	namespacedName := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	if err := fc.Get(ctx, namespacedName, obj); err == nil {
		return fmt.Errorf("object %q already exists", obj.GetName())
	}
	fc.remotesyncs[namespacedName] = *obj.(*gitopsv1alpha1.RemoteSync)

	return nil
}

func (fc *fakeRemoteSyncClient) Delete(_ context.Context, obj client.Object, _ ...client.DeleteOption) error {
	fc.actions = append(fc.actions, fmt.Sprintf("deleting object named %q", obj.GetName()))

	namespacedName := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	delete(fc.remotesyncs, namespacedName)

	return nil
}

func (fc *fakeRemoteSyncClient) Update(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
	fc.actions = append(fc.actions, fmt.Sprintf("updating object named %q", obj.GetName()))

	namespacedName := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	fc.remotesyncs[namespacedName] = *obj.(*gitopsv1alpha1.RemoteSync)

	return nil
}

func (fc *fakeRemoteSyncClient) List(_ context.Context, obj client.ObjectList, _ ...client.ListOption) error {
	fc.actions = append(fc.actions, fmt.Sprintf("listing objects"))

	var remoteSyncList []gitopsv1alpha1.RemoteSync
	for _, rs := range fc.remotesyncs {
		remoteSyncList = append(remoteSyncList, rs)
	}
	*obj.(*gitopsv1alpha1.RemoteSyncList) = gitopsv1alpha1.RemoteSyncList{Items: remoteSyncList}

	return nil
}

func (fc *fakeRemoteSyncClient) Get(_ context.Context, namespacedName types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
	fc.actions = append(fc.actions, fmt.Sprintf("getting object named %q", namespacedName.Name))

	rs, found := fc.remotesyncs[namespacedName]
	if found {
		*obj.(*gitopsv1alpha1.RemoteSync) = rs
		return nil
	}

	return &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
}

func (fc *fakeRemoteSyncClient) setSyncStatus(namespacedName types.NamespacedName, syncStatus string) error {
	rs, found := fc.remotesyncs[namespacedName]
	if found {
		rs.Status.SyncStatus = syncStatus
		fc.remotesyncs[namespacedName] = rs
		return nil
	}

	return &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
}
