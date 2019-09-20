// Copyright 2019 Google LLC
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

package kio_test

import (
	"bytes"
	"log"
	"os"

	"lib.kpt.dev/kio"
	"lib.kpt.dev/yaml"
)

func Example() {
	input := bytes.NewReader([]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  selector:
    app: nginx
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
`))

	// setAnnotationFn
	setAnnotationFn := kio.FilterFunc(func(operand []*yaml.RNode) ([]*yaml.RNode, error) {
		for i := range operand {
			resource := operand[i]
			_, err := resource.Pipe(yaml.SetAnnotation("foo", "bar"))
			if err != nil {
				return nil, err
			}
		}
		return operand, nil
	})

	err := kio.Pipeline{
		Inputs:  []kio.Reader{kio.ByteReader{Reader: input}},
		Filters: []kio.Filter{setAnnotationFn},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: os.Stdout}},
	}.Execute()
	if err != nil {
		log.Fatal(err)
	}

	// Output:
	// apiVersion: apps/v1
	// kind: Deployment
	// metadata:
	//   name: nginx
	//   labels:
	//     app: nginx
	//   annotations:
	//     foo: bar
	// spec:
	//   replicas: 3
	//   selector:
	//     matchLabels:
	//       app: nginx
	//   template:
	//     metadata:
	//       labels:
	//         app: nginx
	//     spec:
	//       containers:
	//       - name: nginx
	//         image: nginx:1.7.9
	//         ports:
	//         - containerPort: 80
	// ---
	// apiVersion: v1
	// kind: Service
	// metadata:
	//   name: nginx
	//   annotations:
	//     foo: bar
	// spec:
	//   selector:
	//     app: nginx
	//   ports:
	//   - protocol: TCP
	//     port: 80
	//     targetPort: 80
}
