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

package types

import (
	"fmt"
	"path"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/cmdexport/pathutil"
)

// PipelineConfig describes configuration of a pipeline.
type PipelineConfig struct {
	Dir     string
	FnPaths []string
	// Current working directory.
	CWD string
}

// UseRelativePaths converts all paths to relative paths to the current working directory.
func (config *PipelineConfig) UseRelativePaths() (err error) {
	config.Dir, err = pathutil.Rel(config.CWD, config.Dir, config.CWD)
	if err != nil {
		return err
	}

	var relativeFnPaths []string
	for _, fnPath := range config.FnPaths {
		fnPath, err = pathutil.Rel(config.CWD, fnPath, config.CWD)
		if err != nil {
			return err
		}

		relativeFnPaths = append(relativeFnPaths, fnPath)
	}
	config.FnPaths = relativeFnPaths

	return nil
}

// checkPaths checks if fnPaths are within the current directory.
func (config *PipelineConfig) CheckFnPaths() (err error) {
	var invalidPaths []string

	for _, fnPath := range config.FnPaths {
		// fnPath might be outside cwd here, e.g. `../functions`
		absoluteFnPath := fnPath
		if !path.IsAbs(fnPath) {
			absoluteFnPath = path.Join(config.CWD, fnPath)
		}

		within, err := pathutil.IsInsideDir(absoluteFnPath, config.CWD)
		if err != nil {
			return err
		}
		if !within {
			invalidPaths = append(invalidPaths, fnPath)
		}
	}

	if len(invalidPaths) > 0 {

		err = fmt.Errorf(
			"function paths are not within the current working directory:\n%s",
			strings.Join(invalidPaths, "\n"),
		)
	}

	return
}
