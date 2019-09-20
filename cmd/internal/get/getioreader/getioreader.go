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

package getioreader

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"lib.kpt.dev/fmtr"

	"kpt.dev/internal/pkgfile"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/yaml"
)

// Get reads a package from input and applies a pattern for generating filesnames.
func Get(path, pattern string, input io.Reader) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}

	b := &bytes.Buffer{}
	fs := &kio.FileSetter{FilenamePattern: pattern, Mode: fmt.Sprintf("%d", 0600)}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{kio.ByteReader{Reader: input}},
		Filters: []kio.Filter{fs, fmtr.Formatter{}},
		Outputs: []kio.Writer{
			kio.ByteWriter{Writer: b, KeepReaderAnnotations: true},
			kio.LocalPackageWriter{PackagePath: path},
		},
	}.Execute()
	if err != nil {
		return err
	}

	k := pkgfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			Kind:       "Kptfile",
			ObjectMeta: yaml.ObjectMeta{Name: filepath.Base(path)},
		},
		Upstream: pkgfile.Upstream{
			Type:  pkgfile.StdinOrigin,
			Stdin: pkgfile.Stdin{Original: b.String(), FilenamePattern: fs.FilenamePattern},
		},
	}
	f, err := os.OpenFile(filepath.Join(path, pkgfile.KptFileName),
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	e := yaml.NewEncoder(f)
	defer e.Close()
	return e.Encode(k)
}
