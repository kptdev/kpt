// Copyright 2022 Google LLC
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

package porch

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoogleContainerTools/kpt/porch/pkg/engine"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"go.opentelemetry.io/otel/trace"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
)

// Watch supports watching for changes.
func (r *packageRevisionResources) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	// 'label' selects on labels; 'field' selects on the object's fields. Not all fields
	// are supported; an error should be returned if 'field' tries to select on a field that
	// isn't supported. 'resourceVersion' allows for continuing/starting a watch at a
	// particular version.

	ctx, span := tracer.Start(ctx, "packageRevisionResources::Watch", trace.WithAttributes())
	defer span.End()

	filter, err := parsePackageRevisionResourcesFieldSelector(options.FieldSelector)
	if err != nil {
		return nil, err
	}

	if ns, namespaced := genericapirequest.NamespaceFrom(ctx); namespaced {
		if filter.Namespace != "" && ns != filter.Namespace {
			return nil, fmt.Errorf("conflicting namespaces specified: %q and %q", ns, filter.Namespace)
		}
		filter.Namespace = ns
	}

	ctx, cancel := context.WithCancel(ctx)

	w := &watcher{
		cancel:     cancel,
		resultChan: make(chan watch.Event, 64),
	}

	go w.listAndWatch(ctx, r, filter, options.LabelSelector)

	return w, nil
}

// watcher implements watch.Interface, and holds the state for an active watch.
type watcher struct {
	cancel func()

	resultChan chan watch.Event

	mutex         sync.Mutex
	eventCallback func(eventType watch.EventType, pr repository.PackageRevision) bool
}

var _ watch.Interface = &watcher{}

// Stop stops watching. Will close the channel returned by ResultChan(). Releases
// any resources used by the watch.
func (w *watcher) Stop() {
	w.cancel()
}

// ResultChan returns a chan which will receive all the events. If an error occurs
// or Stop() is called, the implementation will close this channel and
// release any resources used by the watch.
func (w *watcher) ResultChan() <-chan watch.Event {
	return w.resultChan
}

// listAndWatch implements watch by doing a list, then sending any observed changes.
// This is not a compliant implementation of watch, but it is a good-enough start for most controllers.
// One trick is that we start the watch _before_ we perform the list, so we don't miss changes that happen immediately after the list.
func (w *watcher) listAndWatch(ctx context.Context, r *packageRevisionResources, filter packageRevisionFilter, selector labels.Selector) {
	if err := w.listAndWatchInner(ctx, r, filter, selector); err != nil {
		// TODO: We need to populate the object on this error
		klog.Warningf("sending error to watch stream")
		ev := watch.Event{
			Type: watch.Error,
		}
		w.resultChan <- ev
	}
	w.cancel()
	close(w.resultChan)
}

func (w *watcher) listAndWatchInner(ctx context.Context, r *packageRevisionResources, filter packageRevisionFilter, selector labels.Selector) error {
	errorResult := make(chan error, 4)
	done := false

	var backlog []watch.Event
	w.eventCallback = func(eventType watch.EventType, pr repository.PackageRevision) bool {
		if done {
			return false
		}
		obj, err := pr.GetResources(ctx)
		if err != nil {
			done = true
			errorResult <- err
			return false
		}

		backlog = append(backlog, watch.Event{
			Type:   eventType,
			Object: obj,
		})

		return true
	}
	klog.Infof("starting watch before listing")
	if err := r.packageCommon.watchPackages(ctx, filter, w); err != nil {
		return err
	}

	// TODO: Only if rv == 0?
	if err := r.packageCommon.listPackageRevisions(ctx, filter, selector, func(p *engine.PackageRevision) error {
		obj, err := p.GetResources(ctx)
		if err != nil {
			done = true
			return err
		}
		// TODO: Check resource version?
		ev := watch.Event{
			Type:   watch.Added,
			Object: obj,
		}
		w.sendWatchEvent(ev)
		return nil
	}); err != nil {
		done = true
		return err
	}
	klog.Infof("finished list")

	// Repeatedly flush the backlog until we catch up
	for {
		w.mutex.Lock()
		chunk := backlog
		backlog = nil
		w.mutex.Unlock()

		if len(chunk) == 0 {
			break
		}

		klog.Infof("flushing backlog chunk of length %d", len(chunk))

		for _, ev := range chunk {
			// TODO: Check resource version?

			w.sendWatchEvent(ev)
		}
	}

	w.mutex.Lock()
	// Pick up anything that squeezed in
	for _, ev := range backlog {
		// TODO: Check resource version?

		w.sendWatchEvent(ev)
	}

	klog.Infof("moving watch into streaming mode")
	w.eventCallback = func(eventType watch.EventType, pr repository.PackageRevision) bool {
		if done {
			return false
		}
		obj, err := pr.GetResources(ctx)
		if err != nil {
			done = true
			errorResult <- err
			return false
		}
		// TODO: Check resource version?
		ev := watch.Event{
			Type:   eventType,
			Object: obj,
		}
		w.sendWatchEvent(ev)
		return true
	}
	w.mutex.Unlock()

	select {
	case <-ctx.Done():
		done = true
		return ctx.Err()

	case err := <-errorResult:
		done = true
		return err
	}

}

func (w *watcher) sendWatchEvent(ev watch.Event) {
	// TODO: Handle the case that the watch channel is full?
	klog.Infof("sending watch event %v", ev)
	w.resultChan <- ev
}

// OnPackageRevisionChange is the callback called when a PackageRevision changes.
func (w *watcher) OnPackageRevisionChange(eventType watch.EventType, pr repository.PackageRevision) bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	return w.eventCallback(eventType, pr)
}
