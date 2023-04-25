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

package pull

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/wasmdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/pkg/wasm"
	"github.com/spf13/cobra"
)

const (
	command = "cmdwasmpull"
)

func newRunner(ctx context.Context) *runner {
	r := &runner{
		ctx: ctx,
	}
	c := &cobra.Command{
		Use:     "pull [IMAGE] [LOCAL_PATH]",
		Short:   wasmdocs.PullShort,
		Long:    wasmdocs.PullShort + "\n" + wasmdocs.PullLong,
		Example: wasmdocs.PullExamples,
		RunE:    r.runE,
	}
	r.Command = c
	return r
}

func NewCommand(ctx context.Context) *cobra.Command {
	return newRunner(ctx).Command
}

type runner struct {
	ctx           context.Context
	Command       *cobra.Command
	storageClient *wasm.Client
}

func (r *runner) runE(_ *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	if len(args) != 2 {
		return errors.E(op, "2 positional arguments (a OCI image and a local path) required")
	}

	var err error
	if r.storageClient == nil {
		r.storageClient, err = wasm.NewClient(filepath.Join(os.TempDir(), "kpt"))
		if err != nil {
			return err
		}
	}

	wasmImg := args[0]
	fileName := args[1]
	wasmFileReadCloser, err := r.storageClient.LoadWasm(r.ctx, wasmImg)
	if err != nil {
		return err
	}
	defer wasmFileReadCloser.Close()

	data, err := io.ReadAll(wasmFileReadCloser)
	if err != nil {
		return errors.E(op, "unable to read wasm contents")
	}
	err = os.MkdirAll(filepath.Dir(fileName), 0755)
	if err != nil {
		return errors.E(op, "unable to create directory %q: %w", filepath.Dir(fileName), err)
	}
	if err = os.WriteFile(fileName, data, 0666); err != nil {
		return errors.E(op, "unable to write to file", fileName)
	}
	return nil
}
