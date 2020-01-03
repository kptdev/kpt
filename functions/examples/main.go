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

// Package main implements example kpt-functions
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/functions/examples/helloworld"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func main() {
	cmd := &cobra.Command{
		Use:          "config-function",
		SilenceUsage: true, // don't print usage on an error
		RunE:         (&Dispatcher{}).RunE,
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// Dispatcher dispatches to the matching API
type Dispatcher struct {
	// IO hanldes reading / writing Resources
	IO *kio.ByteReadWriter
}

func (d *Dispatcher) RunE(_ *cobra.Command, _ []string) error {
	d.IO = &kio.ByteReadWriter{
		Reader:                os.Stdin,
		Writer:                os.Stdout,
		KeepReaderAnnotations: true,
	}

	return kio.Pipeline{
		Inputs: []kio.Reader{d.IO},
		Filters: []kio.Filter{
			d, // invoke the API
			&filters.MergeFilter{},
			&filters.FileSetter{FilenamePattern: filepath.Join("config", "%n.yaml")},
			&filters.FormatFilter{},
		},
		Outputs: []kio.Writer{d.IO},
	}.Execute()
}

// dispatchTable maps configFunction Kinds to implementations
var dispatchTable = map[string]func() kio.Filter{
	helloworld.Kind: helloworld.Filter,
}

func (d *Dispatcher) Filter(inputs []*yaml.RNode) ([]*yaml.RNode, error) {
	// parse the API meta to find which API is being invoked
	meta, err := d.IO.FunctionConfig.GetMeta()
	if err != nil {
		return nil, err
	}

	// find the implementation for this API
	fn := dispatchTable[meta.Kind]
	if fn == nil {
		return nil, fmt.Errorf("unsupported API type: %s", meta.Kind)
	}

	// dispatch to the implementation
	fltr := fn()

	// initializes the object from the config
	if err := yaml.Unmarshal([]byte(d.IO.FunctionConfig.MustString()), fltr); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		fmt.Fprintf(os.Stderr, "%s\n", d.IO.FunctionConfig.MustString())
		os.Exit(1)
	}
	return fltr.Filter(inputs)
}
