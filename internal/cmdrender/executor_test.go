// Copyright 2021 Google LLC
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

package cmdrender

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"sigs.k8s.io/kustomize/kyaml/kio"
)

func TestDetectPathConflicts(t *testing.T) {
	tests := map[string]struct {
		input       string // input
		expectedErr string // expected result
	}{
		"duplicate": {
			input: `apiVersion: v1
kind: Custom
metadata:
  name: a
  annotations:
    config.kubernetes.io/path: 'my/path/custom.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/custom.yaml'
    config.kubernetes.io/index: '0'
`,
			expectedErr: `resource at path "my/path/custom.yaml" and index "0" already exists`,
		},
		"duplicate with `./` prefix": {
			input: `apiVersion: v1
kind: Custom
metadata:
  name: a
  annotations:
    config.kubernetes.io/path: 'my/path/custom.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: './my/path/custom.yaml'
    config.kubernetes.io/index: '0'
`,
			expectedErr: `resource at path "my/path/custom.yaml" and index "0" already exists`,
		},
		"duplicate path, not index": {
			input: `apiVersion: v1
kind: Custom
metadata:
  name: a
  annotations:
    config.kubernetes.io/path: 'my/path/custom.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/custom.yaml'
    config.kubernetes.io/index: '1'
`,
		},
		"duplicate index, not path": {
			input: `apiVersion: v1
kind: Custom
metadata:
  name: a
  annotations:
    config.kubernetes.io/path: 'my/path/a.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '0'
`,
		},
		"larger number of resources with duplicate": {
			input: `apiVersion: v1
kind: Custom
metadata:
  name: a
  annotations:
    config.kubernetes.io/path: 'my/path/a.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/a.yaml'
    config.kubernetes.io/index: '1'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '1'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '2'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/c.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/c.yaml'
    config.kubernetes.io/index: '1'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '1'
`,
			expectedErr: `resource at path "my/path/b.yaml" and index "1" already exists`,
		},
		"larger number of resources without duplicates": {
			input: `apiVersion: v1
kind: Custom
metadata:
  name: a
  annotations:
    config.kubernetes.io/path: 'my/path/a.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/a.yaml'
    config.kubernetes.io/index: '1'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '1'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '2'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/c.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/c.yaml'
    config.kubernetes.io/index: '1'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '3'
`,
		},
	}
	for _, tc := range tests {
		out := &bytes.Buffer{}
		r := kio.ByteReadWriter{
			Reader:                bytes.NewBufferString(tc.input),
			Writer:                out,
			KeepReaderAnnotations: true,
			OmitReaderAnnotations: true,
		}
		n, err := r.Read()
		if err != nil {
			fmt.Println("1")
			t.FailNow()
		}
		err = detectPathConflicts(n)
		if err != nil && tc.expectedErr == "" {
			t.Errorf("unexpected error %s", err.Error())
			t.FailNow()
		}
		if tc.expectedErr != "" && err == nil {
			t.Errorf("expected error %s", tc.expectedErr)
			t.FailNow()
		}
		if tc.expectedErr != "" && !strings.Contains(err.Error(), tc.expectedErr) {
			t.Errorf("wanted error %s, got %s", tc.expectedErr, err.Error())
			t.FailNow()
		}
	}
}
