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

package main

import (
	"fmt"
	"os"

	"lib.kpt.dev/kio"
	"lib.kpt.dev/yaml"
)

func main() {
	rw := &kio.ByteReadWriter{Reader: os.Stdin, Writer: os.Stdout, KeepReaderAnnotations: true}

	err := kio.Pipeline{
		Inputs: []kio.Reader{rw},
		Filters: []kio.Filter{kio.FilterFunc(func(in []*yaml.RNode) ([]*yaml.RNode, error) {
			for _, r := range in {
				if err := check(r); err != nil {
					return nil, err
				}
			}
			return in, nil
		})},
		Outputs: []kio.Writer{rw},
	}.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func check(r *yaml.RNode) error {
	containers, err := r.Pipe(yaml.Lookup("spec", "template", "spec", "containers"))
	if err != nil {
		s, _ := r.String()
		return fmt.Errorf("%v: %s", err, s)
	}
	if containers == nil {
		return nil
	}
	return containers.VisitElements(func(node *yaml.RNode) error {
		f, err := node.Pipe(yaml.Lookup("resources", "requests", "cpu"))
		if err != nil {
			s, _ := r.String()
			return fmt.Errorf("%v: %s", err, s)
		}
		if f == nil {
			s, _ := node.Field("name").Value.String()
			return fmt.Errorf(
				"cpu-requests missing for container %s -- use 'set cpu-requests %s'", s, s)
		}

		f, err = node.Pipe(yaml.Lookup("resources", "requests", "memory"))
		if err != nil {
			s, _ := r.String()
			return fmt.Errorf("%v: %s", err, s)
		}
		if f == nil {
			s, _ := node.Field("name").Value.String()
			return fmt.Errorf(
				"memory-requests missing for container %s -- use 'set memory-requests %s'", s, s)
		}
		return nil
	})
}
