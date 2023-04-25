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

package rolloutsclient

import (
	"context"
	"fmt"
	"time"

	rolloutsapi "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	coreapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cli-utils/pkg/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Client implements client for the rollouts API.
type Client struct {
	client client.Client
}

func New() (*Client, error) {
	scheme, err := createScheme()
	if err != nil {
		return nil, err
	}

	config := useServerSideThrottling(config.GetConfigOrDie())
	cl, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}
	return &Client{client: cl}, nil
}

func createScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	for _, api := range (runtime.SchemeBuilder{
		rolloutsapi.AddToScheme,
		coreapi.AddToScheme,
		metav1.AddMetaToScheme,
	}) {
		if err := api(scheme); err != nil {
			return nil, err
		}
	}
	return scheme, nil
}

func (rlc *Client) List(ctx context.Context, ns string) (*rolloutsapi.RolloutList, error) {
	var opts []client.ListOption
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
	}

	rollouts := &rolloutsapi.RolloutList{}
	if err := rlc.client.List(ctx, rollouts, opts...); err != nil {
		return nil, err
	}

	return rollouts, nil
}

func (rlc *Client) Get(ctx context.Context, ns, name string) (*rolloutsapi.Rollout, error) {
	if name == "" {
		return nil, fmt.Errorf("must provide rollout name")
	}

	key := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	rollout := &rolloutsapi.Rollout{}
	if err := rlc.client.Get(ctx, key, rollout); err != nil {
		return nil, err
	}

	return rollout, nil
}

func (rlc *Client) Update(ctx context.Context, rollout *rolloutsapi.Rollout) error {
	return rlc.client.Update(ctx, rollout)
}

func useServerSideThrottling(config *rest.Config) *rest.Config {
	// Timeout if the query takes too long
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	enabled, err := flowcontrol.IsEnabled(ctx, config)
	if err != nil {
		klog.Infof("Couldn't gather flow control configuration from the API apiserver (assuming it is not enabled): %v\n", err)
	}

	if enabled {
		config.QPS = -1
		config.Burst = -1
	}

	return config
}
