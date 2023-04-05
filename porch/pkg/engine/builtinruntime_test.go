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

package engine

import (
	"bytes"
	"context"
	"errors"
	"testing"

	fnsdk "github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
)

func TestBuiltinRuntime(t *testing.T) {
	br := newBuiltinRuntime()
	fn := &v1.Function{
		Image: setNamespaceImageAliases[0],
	}
	fr, err := br.GetRunner(context.Background(), fn)
	if err != nil {
		t.Fatalf("unexpected error when getting the runner: %v", err)
	}
	reader := bytes.NewReader([]byte(`apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
  - apiVersion: v1
    kind: ConfigMap
    metadata:
      name: my-cm
      namespace: old
    data:
      foo: bar
functionConfig:
  apiVersion: v1
  kind: ConfigMap
  data:
    namespace: test-ns
`))
	var buf bytes.Buffer
	err = fr.Run(reader, &buf)
	if err != nil {
		t.Fatalf("unexpected error when running the function runner: %v", err)
	}
	rl, err := fnsdk.ParseResourceList(buf.Bytes())
	if err != nil {
		t.Fatalf("can't parse the updated resource list: %v", err)
	}
	if len(rl.Items) != 1 {
		t.Fatalf("expect the updated resource list to have 1 object in items, but got %d", len(rl.Items))
	}
	ns := rl.Items[0].GetNamespace()
	if ns != "test-ns" {
		t.Fatalf("expect the updated namespace to be %v, but got %v", "test-ns", "ns")
	}
}

func TestBuiltinRuntimeNotFound(t *testing.T) {
	br := newBuiltinRuntime()
	funct := &v1.Function{
		Image: "gcr.io/kpt-fn/not-exist:unstable",
	}
	_, err := br.GetRunner(context.Background(), funct)
	var fnNotFoundErr *fn.NotFoundError
	if !errors.As(err, &fnNotFoundErr) {
		t.Fatalf("expect error to be %T, but got %T %v", fnNotFoundErr, err, err)
	}
}
