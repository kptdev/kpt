// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package apply

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/cli-utils/pkg/apply/info"
	"sigs.k8s.io/cli-utils/pkg/apply/poller"
	"sigs.k8s.io/cli-utils/pkg/apply/prune"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApplierBuilder struct {
	// factory is only used to retrieve things that have not been provided explicitly.
	factory                      util.Factory
	invClient                    inventory.Client
	client                       dynamic.Interface
	discoClient                  discovery.CachedDiscoveryInterface
	mapper                       meta.RESTMapper
	restConfig                   *rest.Config
	unstructuredClientForMapping func(*meta.RESTMapping) (resource.RESTClient, error)
	statusPoller                 poller.Poller
}

// NewApplierBuilder returns a new ApplierBuilder.
func NewApplierBuilder() *ApplierBuilder {
	return &ApplierBuilder{
		// Defaults, if any, go here.
	}
}

func (b *ApplierBuilder) Build() (*Applier, error) {
	bx, err := b.finalize()
	if err != nil {
		return nil, err
	}
	return &Applier{
		pruner: &prune.Pruner{
			InvClient: bx.invClient,
			Client:    bx.client,
			Mapper:    bx.mapper,
		},
		statusPoller:  bx.statusPoller,
		invClient:     bx.invClient,
		client:        bx.client,
		openAPIGetter: bx.discoClient,
		mapper:        bx.mapper,
		infoHelper:    info.NewHelper(bx.mapper, bx.unstructuredClientForMapping),
	}, nil
}

func (b *ApplierBuilder) finalize() (*ApplierBuilder, error) {
	bx := *b // make a copy before mutating any fields. Shallow copy is good enough.
	var err error
	if bx.invClient == nil {
		return nil, errors.New("inventory client must be provided")
	}
	if bx.client == nil {
		if bx.factory == nil {
			return nil, fmt.Errorf("a factory must be provided or all other options: %v", err)
		}
		bx.client, err = bx.factory.DynamicClient()
		if err != nil {
			return nil, fmt.Errorf("error getting dynamic client: %v", err)
		}
	}
	if bx.discoClient == nil {
		if bx.factory == nil {
			return nil, fmt.Errorf("a factory must be provided or all other options: %v", err)
		}
		bx.discoClient, err = bx.factory.ToDiscoveryClient()
		if err != nil {
			return nil, fmt.Errorf("error getting discovery client: %v", err)
		}
	}
	if bx.mapper == nil {
		if bx.factory == nil {
			return nil, fmt.Errorf("a factory must be provided or all other options: %v", err)
		}
		bx.mapper, err = bx.factory.ToRESTMapper()
		if err != nil {
			return nil, fmt.Errorf("error getting rest mapper: %v", err)
		}
	}
	if bx.restConfig == nil {
		if bx.factory == nil {
			return nil, fmt.Errorf("a factory must be provided or all other options: %v", err)
		}
		bx.restConfig, err = bx.factory.ToRESTConfig()
		if err != nil {
			return nil, fmt.Errorf("error getting rest config: %v", err)
		}
	}
	if bx.unstructuredClientForMapping == nil {
		if bx.factory == nil {
			return nil, fmt.Errorf("a factory must be provided or all other options: %v", err)
		}
		bx.unstructuredClientForMapping = bx.factory.UnstructuredClientForMapping
	}
	if bx.statusPoller == nil {
		c, err := client.New(bx.restConfig, client.Options{Scheme: scheme.Scheme, Mapper: bx.mapper})
		if err != nil {
			return nil, fmt.Errorf("error creating client: %v", err)
		}
		bx.statusPoller = polling.NewStatusPoller(c, bx.mapper, polling.Options{})
	}
	return &bx, nil
}

func (b *ApplierBuilder) WithFactory(factory util.Factory) *ApplierBuilder {
	b.factory = factory
	return b
}

func (b *ApplierBuilder) WithInventoryClient(invClient inventory.Client) *ApplierBuilder {
	b.invClient = invClient
	return b
}

func (b *ApplierBuilder) WithDynamicClient(client dynamic.Interface) *ApplierBuilder {
	b.client = client
	return b
}

func (b *ApplierBuilder) WithDiscoveryClient(discoClient discovery.CachedDiscoveryInterface) *ApplierBuilder {
	b.discoClient = discoClient
	return b
}

func (b *ApplierBuilder) WithRestMapper(mapper meta.RESTMapper) *ApplierBuilder {
	b.mapper = mapper
	return b
}

func (b *ApplierBuilder) WithRestConfig(restConfig *rest.Config) *ApplierBuilder {
	b.restConfig = restConfig
	return b
}

func (b *ApplierBuilder) WithUnstructuredClientForMapping(unstructuredClientForMapping func(*meta.RESTMapping) (resource.RESTClient, error)) *ApplierBuilder {
	b.unstructuredClientForMapping = unstructuredClientForMapping
	return b
}

func (b *ApplierBuilder) WithStatusPoller(statusPoller poller.Poller) *ApplierBuilder {
	b.statusPoller = statusPoller
	return b
}
