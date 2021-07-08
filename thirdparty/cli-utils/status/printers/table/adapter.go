// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/print/table"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	pe "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// CollectorAdapter wraps the ResourceStatusCollector and
// provides a set of functions that matches the interfaces
// needed by the BaseTablePrinter.
type CollectorAdapter struct {
	collector *collector.ResourceStatusCollector
}

type ResourceInfo struct {
	resourceStatus *pe.ResourceStatus
}

func (r *ResourceInfo) Identifier() object.ObjMetadata {
	return r.resourceStatus.Identifier
}

func (r *ResourceInfo) ResourceStatus() *pe.ResourceStatus {
	return r.resourceStatus
}

func (r *ResourceInfo) SubResources() []table.Resource {
	var subResources []table.Resource
	for _, rs := range r.resourceStatus.GeneratedResources {
		subResources = append(subResources, &ResourceInfo{
			resourceStatus: rs,
		})
	}
	return subResources
}

type ResourceState struct {
	resources []table.Resource
	err       error
}

func (rss *ResourceState) Resources() []table.Resource {
	return rss.resources
}

func (rss *ResourceState) Error() error {
	return rss.err
}

func (ca *CollectorAdapter) LatestStatus() *ResourceState {
	observation := ca.collector.LatestObservation()
	var resources []table.Resource
	for _, resourceStatus := range observation.ResourceStatuses {
		resources = append(resources, &ResourceInfo{
			resourceStatus: resourceStatus,
		})
	}
	return &ResourceState{
		resources: resources,
		err:       observation.Error,
	}
}
