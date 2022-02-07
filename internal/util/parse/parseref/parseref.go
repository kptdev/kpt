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

package parseref

import (
	"fmt"
	"os"
	"path/filepath"

	kpterrors "github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/pkg/location"
)

func ParseArgs(args []string, opts ...location.Option) (location.Reference, string, error) {
	const op kpterrors.Op = "parse.ParseArgs"

	ref, err := location.ParseReference(args[0], opts...)
	if err != nil {
		return nil, "", kpterrors.E(op, err)
	}

	if args[1] == "" {
		return ref, "", nil
	}

	dir, err := getDest(args[1], ref)
	if err != nil {
		return nil, "", kpterrors.E(op, err)
	}

	return ref, dir, nil
}

func getDest(dir string, ref location.Reference) (string, error) {
	destination := filepath.Clean(dir)

	f, err := os.Stat(destination)
	if os.IsNotExist(err) {
		parent := filepath.Dir(destination)
		if _, err := os.Stat(parent); os.IsNotExist(err) {
			// error -- fetch to directory where parent does not exist
			return "", fmt.Errorf("parent directory %q does not exist", parent)
		}
		// fetch to a specific directory -- don't default the name
		return destination, nil
	}

	if !f.IsDir() {
		return "", fmt.Errorf("LOCAL_PKG_DEST must be a directory")
	}

	if name, ok := location.DefaultDirectoryName(ref); ok {
		return filepath.Join(destination, name), nil
	}

	// this reference type does not provide a default name.
	// the error message is a prompt to provide complete path to new dir.
	return "", fmt.Errorf("destination directory already exists")
}
