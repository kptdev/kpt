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

package push

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/wasmdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/pkg/wasm"
	"github.com/spf13/cobra"
)

const (
	command = "cmdwasmpush"
)

func newRunner(ctx context.Context) *runner {
	r := &runner{
		ctx: ctx,
	}
	c := &cobra.Command{
		Use:     "push [LOCAL_PATH] [IMAGE]",
		Short:   wasmdocs.PushShort,
		Long:    wasmdocs.PushShort + "\n" + wasmdocs.PushLong,
		Example: wasmdocs.PushExamples,
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
		return errors.E(op, "2 positional arguments (local wasm file and OCI image) are required")
	}

	var err error
	if r.storageClient == nil {
		r.storageClient, err = wasm.NewClient(path.Join(os.TempDir(), "kpt"))
		if err != nil {
			return err
		}
	}

	wasmFile := args[0]
	img := args[1]

	err = r.storageClient.PushWasm(r.ctx, wasmFile, img)
	if err != nil {
		return err
	}

	fmt.Printf("image has been pushed to %v\n", img)
	return nil
}
