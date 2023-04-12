// Copyright 2020 The kpt Authors
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

package util

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/util/cfgflags"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	cluster "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/flowcontrol"
)

func NewFactory(cmd *cobra.Command, version string) cluster.Factory {
	flags := cmd.PersistentFlags()
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).
		WithDeprecatedPasswordFlag()
	kubeConfigFlags.AddFlags(flags)
	UpdateQPS(kubeConfigFlags)
	userAgentKubeConfigFlags := &cfgflags.UserAgentKubeConfigFlags{
		Delegate:  kubeConfigFlags,
		UserAgent: fmt.Sprintf("kpt/%s", version),
	}
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return cluster.NewFactory(userAgentKubeConfigFlags)
}

// UpdateQPS modifies a genericclioptions.ConfigFlags to update the client-side
// throttling QPS and Burst QPS (including for discovery).
//
// If Flow Control is enabled on the apiserver, client-side throttling is
// disabled!
//
// If Flow Control is disabled or undetected on the apiserver, client-side
// throttling QPS will be increased to at least 30 (burst: 60).
//
// Flow Control is enabled by default on Kubernetes v1.20+.
// https://kubernetes.io/docs/concepts/cluster-administration/flow-control/
func UpdateQPS(flags *genericclioptions.ConfigFlags) {
	flags.
		WithWrapConfigFn(func(c *rest.Config) *rest.Config {
			// Timeout if the query takes too long, defaulting to the lower QPS limits.
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			enabled, err := flowcontrol.IsEnabled(ctx, c)
			if err != nil {
				klog.Warning("Failed to query apiserver to check for flow control enablement: %v", err)
				// Default to the lower QPS limits.
			}

			qps := float32(-1)
			burst := -1
			if enabled {
				klog.V(1).Infof("Flow control enabled on apiserver: client-side throttling QPS set to %.0f (burst: %d)", qps, burst)
			} else {
				qps = maxIfNotNegative(c.QPS, 30)
				burst = int(maxIfNotNegative(float32(c.Burst), 60))
				klog.V(1).Infof("Flow control disabled on apiserver: client-side throttling QPS set to %.0f (burst: %d)", qps, burst)
			}

			c.QPS = qps
			c.Burst = burst
			flags.
				WithDiscoveryQPS(qps).
				WithDiscoveryBurst(burst)

			return c
		})
}

func maxIfNotNegative(a, b float32) float32 {
	switch {
	case a < 0:
		return a
	case a > b:
		return a
	default:
		return b
	}
}
