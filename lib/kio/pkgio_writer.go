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

package kio

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"lib.kpt.dev/yaml"
)

// LocalPackageWriter writes ResourceNodes to a filesystem
type LocalPackageWriter struct {
	Kind string `yaml:"kind,omitempty"`

	// PackagePath is the path to the package directory.
	PackagePath string `yaml:"path,omitempty"`

	// KeepReaderAnnotations if set will retain the annotations set by LocalPackageReader
	KeepReaderAnnotations bool `yaml:"keepReaderAnnotations,omitempty"`

	// ClearAnnotations will clear annotations before writing the resources
	ClearAnnotations []string `yaml:"clearAnnotations,omitempty"`
}

var _ Writer = LocalPackageWriter{}

func (r LocalPackageWriter) Write(nodes []*yaml.RNode) error {
	if err := ErrorIfMissingAnnotation(nodes, requiredResourcePackageAnnotations...); err != nil {
		return err
	}

	if s, err := os.Stat(r.PackagePath); err != nil {
		return err
	} else if !s.IsDir() {
		// if the user specified input isn't a directory, the package is the directory of the
		// target
		r.PackagePath = filepath.Dir(r.PackagePath)
	}

	// setup indexes for writing Resources back to files
	if err := r.errorIfMissingRequiredAnnotation(nodes); err != nil {
		return err
	}
	outputFiles, outputModes, err := r.indexByFilePath(nodes)
	if err != nil {
		return err
	}
	for k := range outputFiles {
		if err = sortNodes(outputFiles[k]); err != nil {
			return err
		}
	}

	if !r.KeepReaderAnnotations {
		r.ClearAnnotations = append(r.ClearAnnotations, ModeAnnotation)
		r.ClearAnnotations = append(r.ClearAnnotations, PackageAnnotation)
		r.ClearAnnotations = append(r.ClearAnnotations, PathAnnotation)
	}

	// validate outputs before writing any
	for path := range outputFiles {
		outputPath := filepath.Join(r.PackagePath, path)
		if st, err := os.Stat(outputPath); !os.IsNotExist(err) {
			if err != nil {
				return err
			}
			if st.IsDir() {
				return fmt.Errorf("kpt.dev/kio/path cannot be a directory: %s", path)
			}
		}

		err = os.MkdirAll(filepath.Dir(outputPath), 0700)
		if err != nil {
			return err
		}

		if _, found := outputModes[path]; !found {
			// this should never happen - coding error
			return fmt.Errorf("no output mode for %s", path)
		}
	}

	// write files
	for path := range outputFiles {
		outputPath := filepath.Join(r.PackagePath, path)
		err = os.MkdirAll(filepath.Dir(filepath.Join(r.PackagePath, path)), 0700)
		if err != nil {
			return err
		}

		f, err := os.OpenFile(
			outputPath,
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			os.FileMode(outputModes[path]),
		)
		if err != nil {
			return err
		}
		if err := func() error {
			defer f.Close()
			w := ByteWriter{
				Writer:                f,
				KeepReaderAnnotations: r.KeepReaderAnnotations,
				ClearAnnotations:      r.ClearAnnotations,
			}
			if err = w.Write(outputFiles[path]); err != nil {
				return err
			}
			return nil
		}(); err != nil {
			return err
		}
	}

	return nil
}

func (r LocalPackageWriter) errorIfMissingRequiredAnnotation(nodes []*yaml.RNode) error {
	for i := range nodes {
		for _, s := range requiredResourcePackageAnnotations {
			key, err := nodes[i].Pipe(yaml.GetAnnotation(s))
			if err != nil {
				return err
			}
			if key == nil || key.YNode() == nil || key.YNode().Value == "" {
				return fmt.Errorf(
					"resources must be annotated with %s to be written to files", s)
			}
		}
	}
	return nil
}

func (r LocalPackageWriter) indexByFilePath(
	nodes []*yaml.RNode) (map[string][]*yaml.RNode,
	map[string]int, error) {

	outputFiles := map[string][]*yaml.RNode{}
	outputModes := map[string]int{}
	for i := range nodes {
		// parse the file write path
		node := nodes[i]
		value, err := node.Pipe(yaml.GetAnnotation(PathAnnotation))
		if err != nil {
			// this should never happen if errorIfMissingRequiredAnnotation was run
			return nil, nil, err
		}
		path := value.YNode().Value
		outputFiles[path] = append(outputFiles[path], node)

		if filepath.IsAbs(path) {
			return nil, nil, fmt.Errorf("package paths may not be absolute paths")
		}
		if strings.Contains(filepath.Clean(path), "..") {
			return nil, nil, fmt.Errorf("resource must be written under package %s: %s",
				r.PackagePath, filepath.Clean(path))
		}

		// parse the file write mode
		value, err = node.Pipe(yaml.GetAnnotation(ModeAnnotation))
		if err != nil {
			// this should never happen if errorIfMissingRequiredAnnotation was run
			return nil, nil, err
		}
		mode, err := strconv.Atoi(value.YNode().Value)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to parse %s: %v", ModeAnnotation, err)
		}
		if m, found := outputModes[path]; found {
			if m != mode {
				return nil, nil, fmt.Errorf("conflicting file modes %d %d", m, mode)
			}
		} else {
			outputModes[path] = mode
		}
	}
	return outputFiles, outputModes, nil
}
