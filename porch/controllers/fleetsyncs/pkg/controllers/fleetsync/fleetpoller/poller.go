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

package fleetpoller

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/controllers/fleetsyncs/api/v1alpha1"
	gkehubv1 "google.golang.org/api/gkehub/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func NewPoller(channel chan event.GenericEvent) *Poller {
	return &Poller{
		channel:    channel,
		projectIds: make(map[string][]types.NamespacedName),
	}
}

type Poller struct {
	channel    chan event.GenericEvent
	cancelFunc context.CancelFunc

	projectIds map[string][]types.NamespacedName
	pollResult map[string]*PollResult
	mutex      sync.Mutex
}

func (p *Poller) Start() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	p.cancelFunc = cancelFunc
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for {
			select {
			case <-ticker.C:
				p.pollOnce(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (p *Poller) VerifyProjectIdsForFleetSync(fleetSync types.NamespacedName, projectIds []string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	// This is not a very efficient way to do this...
	for projectId, nns := range p.projectIds {
		var newNns []types.NamespacedName
		for _, nn := range nns {
			if nn != fleetSync {
				newNns = append(newNns, nn)
			}
		}
		p.projectIds[projectId] = newNns
	}

	for _, projectId := range projectIds {
		if nns, found := p.projectIds[projectId]; !found {
			p.projectIds[projectId] = []types.NamespacedName{fleetSync}
		} else {
			p.projectIds[projectId] = append(nns, fleetSync)
		}
	}

	klog.Infof("projectIds count %d", len(p.projectIds))
	for projectId := range p.projectIds {
		klog.Infof("ProjectId: %s", projectId)
	}
}

func (p *Poller) StopPollingForFleetSync(fleetSync types.NamespacedName) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	// This is not a very efficient way to do this...
	for projectId, nns := range p.projectIds {
		var newNns []types.NamespacedName
		for _, nn := range nns {
			if nn != fleetSync {
				newNns = append(newNns, nn)
			}
		}
		p.projectIds[projectId] = newNns
	}
}

func (p *Poller) LatestResult(projectId string) (*PollResult, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	res, found := p.pollResult[projectId]
	if !found {
		return nil, false
	}
	return res, true
}

type PollResult struct {
	MembershipsErr error
	Memberships    []*gkehubv1.Membership

	ScopesErr error
	Scopes    []*gkehubv1.Scope

	BindingsErrs []error
	Bindings     []*gkehubv1.MembershipBinding
}

func (pr *PollResult) HasError() bool {
	return pr.MembershipsErr != nil || pr.ScopesErr != nil || len(pr.BindingsErrs) > 0
}

func (pr *PollResult) ErrorSummary() error {
	if !pr.HasError() {
		return nil
	}

	var builder strings.Builder
	builder.WriteString("Errors:")
	if pr.MembershipsErr != nil {
		builder.WriteString(fmt.Sprintf(" [memberships: %s]", pr.MembershipsErr.Error()))
	}
	if pr.ScopesErr != nil {
		builder.WriteString(fmt.Sprintf(" [scopes: %s]", pr.ScopesErr.Error()))
	}
	if len(pr.BindingsErrs) > 0 {
		builder.WriteString(fmt.Sprintf(" [bindings: %d errors", len(pr.BindingsErrs)))
		for _, err := range pr.BindingsErrs {
			builder.WriteString(fmt.Sprintf(", %s", err.Error()))
		}
	}
	return fmt.Errorf(builder.String())
}

func (p *Poller) pollOnce(ctx context.Context) {
	var projectIds map[string][]types.NamespacedName
	var previousPollResult map[string]*PollResult
	func() {
		p.mutex.Lock()
		defer p.mutex.Unlock()
		projectIds = p.projectIds
		previousPollResult = p.pollResult
	}()

	klog.Infof("Polling %d projects", len(projectIds))

	newPollResult := p.poll(ctx, projectIds)

	toNotify := make(map[types.NamespacedName]struct{})
	for projectId, newRes := range newPollResult {
		klog.Infof("Checking for changes for projectId %s", projectId)
		oldRes, found := previousPollResult[projectId]
		// No result from a previous run means it must have been
		// added later. Schedule a reconcile for all FleetSyncs
		// referencing the projectId.
		if !found {
			klog.Infof("Not found")
			nns := projectIds[projectId]
			for _, nn := range nns {
				toNotify[nn] = struct{}{}
			}
			continue
		}
		// If either the previous poll or the current poll errored
		// out, trigger a reconcile.
		if newRes.HasError() || oldRes.HasError() {
			klog.Infof("Has errors")
			nns := projectIds[projectId]
			for _, nn := range nns {
				toNotify[nn] = struct{}{}
			}
			continue
		}

		// If any of the memberships have changed, trigger a reconcile.
		if !equality.Semantic.DeepEqual(newRes, oldRes) {
			klog.Infof("Not equal")
			nns := projectIds[projectId]
			for _, nn := range nns {
				toNotify[nn] = struct{}{}
			}
		}
	}

	func() {
		p.mutex.Lock()
		defer p.mutex.Unlock()
		p.pollResult = newPollResult
	}()

	// Notify after we have updated the poll result, so any triggered
	// reconcile will see the latest data.
	for nn := range toNotify {
		klog.Infof("Triggering reconcile for %s", nn.String())
		fs := &v1alpha1.FleetSync{}
		fs.SetName(nn.Name)
		fs.SetNamespace(nn.Namespace)
		p.channel <- event.GenericEvent{
			Object: fs,
		}
	}
}

func (p *Poller) poll(ctx context.Context, projectIds map[string][]types.NamespacedName) map[string]*PollResult {
	res := make(map[string]*PollResult)
	for projectId := range projectIds {
		pr := &PollResult{}
		res[projectId] = pr
		respM, err := p.listMemberships(ctx, projectId)
		if err != nil {
			pr.MembershipsErr = err
			klog.Infof("Membership polling failed: %v", err)
		} else {
			pr.Memberships = respM.Resources
			klog.Infof("Polling %s found %d memberships", projectId, len(respM.Resources))
		}

		respS, err := p.listScopes(ctx, projectId)
		if err != nil {
			pr.ScopesErr = err
			klog.Infof("Scope polling failed: %v", err)
		} else {
			pr.Scopes = respS.Scopes
			klog.Infof("Polling %s found %d scopes", projectId, len(respS.Scopes))
		}

		if pr.MembershipsErr != nil {
			pr.BindingsErrs = []error{fmt.Errorf("Could not list bindings due to membership retrieval error")}
			continue
		}

		for _, mbr := range pr.Memberships {
			respMB, err := p.listMembershipBindings(ctx, mbr.Name)
			if err != nil {
				pr.BindingsErrs = append(pr.BindingsErrs, err)
				klog.Infof("MembershipBinding polling failed: %v", err)
			} else {
				pr.Bindings = append(pr.Bindings, respMB.MembershipBindings...)
				klog.Infof("Polling %s found %d membership bindings", projectId, len(respMB.MembershipBindings))
			}
		}

	}
	return res
}

func (p *Poller) listMemberships(ctx context.Context, projectId string) (*gkehubv1.ListMembershipsResponse, error) {
	hubClient, err := gkehubv1.NewService(ctx)
	if err != nil {
		return nil, err
	}

	parent := fmt.Sprintf("projects/%s/locations/global", projectId)
	resp, err := hubClient.Projects.Locations.Memberships.List(parent).Do()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (p *Poller) listScopes(ctx context.Context, projectId string) (*gkehubv1.ListScopesResponse, error) {
	hubClient, err := gkehubv1.NewService(ctx)
	if err != nil {
		return nil, err
	}

	parent := fmt.Sprintf("projects/%s/locations/global", projectId)
	resp, err := hubClient.Projects.Locations.Scopes.List(parent).Do()
	if err != nil {
		return nil, err
	}

	return resp, nil
}
func (p *Poller) listMembershipBindings(ctx context.Context, membership string) (*gkehubv1.ListMembershipBindingsResponse, error) {
	hubClient, err := gkehubv1.NewService(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := hubClient.Projects.Locations.Memberships.Bindings.List(membership).Do()
	if err != nil {
		return nil, err
	}

	return resp, nil
}
