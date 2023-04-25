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

package get

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/commands/alpha/rollouts/rolloutsclient"
	rolloutsapi "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	k8scmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func NewCommand(ctx context.Context, f k8scmdutil.Factory) *cobra.Command {
	return newRunner(ctx, f).Command
}

func newRunner(ctx context.Context, f k8scmdutil.Factory) *runner {
	r := &runner{
		ctx:     ctx,
		factory: f,
	}
	c := &cobra.Command{
		Use:     "get",
		Short:   "lists rollouts",
		Long:    "lists rollouts",
		Example: "lists rollouts",
		RunE:    r.runE,
	}
	r.Command = c
	return r
}

type runner struct {
	ctx     context.Context
	Command *cobra.Command
	factory k8scmdutil.Factory
}

func (r *runner) runE(cmd *cobra.Command, _ []string) error {
	rlc, err := rolloutsclient.New()
	if err != nil {
		fmt.Printf("%s\n", err)
		return err
	}

	namespace, _, err := r.factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	rollouts, err := rlc.List(r.ctx, namespace)
	if err != nil {
		fmt.Printf("%s\n", err)
		return err
	}
	renderRolloutsAsTable(cmd, rollouts)
	return nil
}

func renderRolloutsAsTable(cmd *cobra.Command, rollouts *rolloutsapi.RolloutList) {
	t := table.NewWriter()
	t.SetOutputMirror(cmd.OutOrStdout())
	t.AppendHeader(table.Row{"ROLLOUT", "STATUS", "CLUSTERS (READY/TOTAL)"})
	for _, rollout := range rollouts.Items {
		readyCount := 0
		for _, cluster := range rollout.Status.ClusterStatuses {
			if cluster.PackageStatus.Status == "Synced" {
				readyCount++
			}
		}
		t.AppendRow([]interface{}{
			rollout.Name,
			rollout.Status.Overall,
			fmt.Sprintf("%d/%d", readyCount, len(rollout.Status.ClusterStatuses))})
	}
	t.AppendSeparator()
	t.Render()
}
