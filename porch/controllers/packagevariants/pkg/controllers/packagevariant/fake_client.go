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

package packagevariant

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type fakeClient struct {
	output []string
	client.Client
}

var _ client.Client = &fakeClient{}

func (f *fakeClient) Create(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
	f.output = append(f.output, fmt.Sprintf("creating object: %s", obj.GetName()))
	return nil
}

func (f *fakeClient) Delete(_ context.Context, obj client.Object, _ ...client.DeleteOption) error {
	f.output = append(f.output, fmt.Sprintf("deleting object: %s", obj.GetName()))
	return nil
}

func (f *fakeClient) Update(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
	f.output = append(f.output, fmt.Sprintf("updating object: %s", obj.GetName()))
	return nil
}

func (f *fakeClient) List(_ context.Context, obj client.ObjectList, _ ...client.ListOption) error {
	cmList := `apiVersion: v1
kind: ConfigMapList
metadata:
  name: my-cm-list
items:
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: us-east1-endpoints
  data:
    db: db.us-east1.example.com
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: us-east2-endpoints
  data:
    db: db.us-east2.example.com
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: us-east3-endpoints
  data:
    db: db.us-east3.example.com`

	teamList := `apiVersion: v1
kind: TeamList
metadata:
  name: my-team-list
items:
- apiVersion: hr.example.com/v1alpha1
  kind: Team
  metadata:
    name: dev-team-alpha
  spec:
    chargeCode: ab
- apiVersion: hr.example.com/v1alpha1
  kind: Team
  metadata:
    name: dev-team-beta
  spec:
    chargeCode: cd
- apiVersion: hr.example.com/v1alpha1
  kind: Team
  metadata:
    name: prod-team
  spec:
    chargeCode: ef`

	var err error
	switch v := obj.(type) {
	case *unstructured.UnstructuredList:
		gvk := v.GroupVersionKind()
		switch gvk.Kind {
		case "Team":
			err = yaml.Unmarshal([]byte(teamList), v)
		case "ConfigMap":
			err = yaml.Unmarshal([]byte(cmList), v)
		default:
			return fmt.Errorf("unsupported kind")
		}
	default:
		return fmt.Errorf("unsupported type")
	}
	return err
}
