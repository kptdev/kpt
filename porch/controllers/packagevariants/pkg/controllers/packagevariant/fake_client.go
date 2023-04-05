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

package packagevariant

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
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
