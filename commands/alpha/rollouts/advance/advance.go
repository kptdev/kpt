// Copyright 2023 The kpt Authors
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

package advance

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/commands/alpha/rollouts/rolloutsclient"
	"github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/spf13/cobra"
	k8scmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func newRunner(ctx context.Context, f k8scmdutil.Factory) *runner {
	r := &runner{
		ctx:     ctx,
		factory: f,
	}
	c := &cobra.Command{
		Use:     "advance rollout-name wave-name",
		Short:   "advances the wave of a progressive rollout",
		Long:    "advances the wave of a progressive rollout",
		Example: "advances the wave of a progressive rollout",
		RunE:    r.runE,
	}
	r.Command = c
	return r
}

func NewCommand(ctx context.Context, f k8scmdutil.Factory) *cobra.Command {
	return newRunner(ctx, f).Command
}

type runner struct {
	ctx     context.Context
	Command *cobra.Command
	factory k8scmdutil.Factory
}

func (r *runner) runE(_ *cobra.Command, args []string) error {
	rlc, err := rolloutsclient.New()
	if err != nil {
		fmt.Printf("%s\n", err)
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("must provide rollout name")
	}

	if len(args) == 1 {
		return fmt.Errorf("must provide wave name")
	}

	rolloutName := args[0]
	waveName := args[1]

	namespace, _, err := r.factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	rollout, err := rlc.Get(r.ctx, namespace, rolloutName)
	if err != nil {
		fmt.Printf("%s\n", err)
		return err
	}

	if rollout.Spec.Strategy.Type != v1alpha1.Progressive {
		return fmt.Errorf("rollout must be using the progressive strategy to use this command")
	}

	if rollout.Status.WaveStatuses != nil {
		waveFound := false

		for _, waveStatus := range rollout.Status.WaveStatuses {
			if waveStatus.Name == waveName {
				waveFound = true
				break
			}
		}

		if !waveFound {
			return fmt.Errorf("wave %q not found in this rollout", waveName)
		}
	}

	rollout.Spec.Strategy.Progressive.PauseAfterWave.WaveName = waveName

	err = rlc.Update(r.ctx, rollout)
	if err != nil {
		fmt.Printf("%s\n", err)
		return err
	}

	fmt.Println("done")
	return nil
}
