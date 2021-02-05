// Copyright 2020 Google LLC
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

package pipeline

import (
	"bytes"
	"fmt"
	"io"

	"k8s.io/klog"
	"sigs.k8s.io/kustomize/kyaml/comments"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// annotation used to track resources for preserving comments during the function execution
	idAnnotation = "config.k8s.io/id"
)

// KRMFn defines capabilities of a KRM function. It primarily knows
// how to run itself. It accepts input from an io.Reader that is
// a stream of KRM objects in the form of KRM FunctionSpec. It also accepts
// a io.Rriter where it writes the result of running a KRM function.
type KRMFn interface {
	Run(r io.Reader, w io.Writer) error
}

// So we will have different types of KRMFn implementation:
// ExecFn: that can execute a binary on local machine
// NetworkFn: that can invoke a endpoint remotely
// ContainerFn: that can execute a container locally
// Built-in functions that are compiled into kpt

// annotator is built kpt function for annotating KRM resources.
type annotator struct{}

func (a *annotator) Run(r io.Reader, w io.Writer) error {
	rw := &kio.ByteReadWriter{
		Reader:                r,
		Writer:                w,
		KeepReaderAnnotations: true,
	}

	items, err := rw.Read()
	if err != nil {
		return err
	}

	fnConfig := rw.FunctionConfig
	if fnConfig == nil {
		return nil
	}
	dataMapsNode, err := fnConfig.Pipe(yaml.Lookup("data"))
	if err != nil {
		return err
	}
	keyValues := make(map[string]string)
	err = dataMapsNode.VisitFields(func(node *yaml.MapNode) error {
		key := node.Key.YNode().Value
		val := node.Value.YNode().Value
		keyValues[key] = val
		return nil
	})
	if err != nil {
		return err
	}

	for i := range items {
		for k, v := range keyValues {
			if err := items[i].PipeE(yaml.SetAnnotation(k, v)); err != nil {
				return err
			}
		}
		klog.Infof("processing file: %v", items[i])
	}
	return rw.Write(items)
}

// fnRunner adapts a given KRMFn into into kio.Filter to that
// it can be run as a filter in a kio.Pipeline. Another way to think
// of it that it knows how to run a KRMFn.
type fnRunner struct {
	fn KRMFn

	// ids book keeping of resources to preserve comments
	// during the transformation.
	ids map[string]*yaml.RNode

	// fnConfig contains the configs for this function
	fnConfig *yaml.RNode
}

func (f *fnRunner) Filter(resources []*yaml.RNode) (output []*yaml.RNode, err error) {
	fn := f.fn

	fnInput := &bytes.Buffer{}
	fnOutput := &bytes.Buffer{}

	err = f.setIds(resources)
	if err != nil {
		return output, err
	}

	//
	// Wrap the input resources in a list and write it in a stream
	// that is fed to the function chains. kio.ByteWriter does that.
	//
	err = kio.ByteWriter{
		WrappingAPIVersion:    kio.ResourceListAPIVersion,
		WrappingKind:          kio.ResourceListKind,
		Writer:                fnInput,
		KeepReaderAnnotations: true,
		FunctionConfig:        f.fnConfig,
	}.Write(resources)
	if err != nil {
		err = fmt.Errorf("failed to write resource list %w", err)
		return output, err
	}

	result := &kio.ByteReader{Reader: fnOutput}

	err = fn.Run(fnInput, fnOutput)
	if err != nil {
		klog.Errorf("failed to execute function: %v", err)
		return output, err
	}

	output, err = result.Read()
	if err != nil {
		klog.Errorf("failed to read the output from fn execution: %v", err)
		return output, err
	}

	err = f.setComments(output)
	if err != nil {
		return output, err
	}

	// TODO annotate any generated Resources with a path and index if they don't already have one
	// if err := kioutil.DefaultPathAnnotation(functionDir, output); err != nil {
	// return nil, err
	// }
	return output, err
}

// Note(droot): code below is copied from kyaml fn runner as is.

func (f *fnRunner) setIds(nodes []*yaml.RNode) error {
	// set the id on each node to map inputs to outputs
	var id int
	f.ids = map[string]*yaml.RNode{}
	for i := range nodes {
		id++
		idStr := fmt.Sprintf("%v", id)
		err := nodes[i].PipeE(yaml.SetAnnotation(idAnnotation, idStr))
		if err != nil {
			return fmt.Errorf("failed to set id annotation %w", err)
		}
		f.ids[idStr] = nodes[i]
	}
	return nil
}

func (f *fnRunner) setComments(nodes []*yaml.RNode) error {
	for i := range nodes {
		node := nodes[i]
		anID, err := node.Pipe(yaml.GetAnnotation(idAnnotation))
		if err != nil {
			return fmt.Errorf("failed to retrieved id annotation %w", err)
		}
		if anID == nil {
			continue
		}

		var in *yaml.RNode
		var found bool
		if in, found = f.ids[anID.YNode().Value]; !found {
			continue
		}
		if err := comments.CopyComments(in, node); err != nil {
			return fmt.Errorf("failed to copy comments %w", err)
		}
		if err := node.PipeE(yaml.ClearAnnotation(idAnnotation)); err != nil {
			return fmt.Errorf("failed to clean id annotation %w", err)
		}
	}
	return nil
}
