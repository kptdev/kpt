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

package controllers

import (
	"context"
	"time"

	"github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

const (
	minReconnectDelay = 1 * time.Second
	maxReconnectDelay = 30 * time.Second
)

type watcher struct {
	clusterRef v1alpha1.ClusterRef

	ctx context.Context

	cancelFunc context.CancelFunc

	getDynamicClient func(ctx context.Context) (dynamic.Interface, error)

	channel chan event.GenericEvent

	liens map[types.NamespacedName]struct{}
}

func (w *watcher) watch() {
	logger := klog.FromContext(w.ctx)
	var events <-chan watch.Event
	var watcher watch.Interface
	var bookmark string
	defer func() {
		if watcher != nil {
			watcher.Stop()
		}
	}()

	reconnect := newBackoffTimer(minReconnectDelay, maxReconnectDelay)
	defer reconnect.Stop()

loop:
	for {
		select {
		case <-reconnect.channel():
			var err error
			logger.Info("Starting watch")
			watcher, err = w.watchResource(w.ctx, rootSyncGVR)
			if err != nil {
				logger.Error(err, "Cannot start watch; will retry", err)
				reconnect.backoff()
			} else {
				logger.Info("Watch successfully started")
				events = watcher.ResultChan()
			}
		case e, ok := <-events:
			if !ok {
				logger.Info("Watch event stream closed; will restart watch from bookmark", "bookmark", bookmark)
				watcher.Stop()
				events = nil
				watcher = nil

				// Initiate reconnect
				reconnect.reset()
			} else if obj, ok := e.Object.(*unstructured.Unstructured); ok {
				if e.Type == watch.Bookmark {
					bookmark = obj.GetResourceVersion()
					logger.Info("Watch bookmark", "bookmark", bookmark)
				} else {
					bookmark = obj.GetResourceVersion()
					logger.Info("Got watch event", "bookmark", bookmark)
					w.channel <- event.GenericEvent{
						Object: obj,
					}
				}
			} else {
				logger.V(5).Info("Received unexpected watch event object", "watchEventObject", e.Object)
			}
		case <-w.ctx.Done():
			if w.ctx.Err() != nil {
				logger.V(2).Info("Exiting watcher for because context is done", "err", w.ctx.Err())
			} else {
				logger.Info("Watch background routine exiting; context done")
			}
			break loop
		}
	}
}

func (w *watcher) watchResource(ctx context.Context, gvr schema.GroupVersionResource) (watch.Interface, error) {
	dynamicClient, err := w.getDynamicClient(ctx)
	if err != nil {
		return nil, err
	}

	return dynamicClient.Resource(gvr).Watch(ctx, v1.ListOptions{})
}

// TODO: This comes from Porch. Find a place to put code that can be shared.
type backoffTimer struct {
	min, max, curr time.Duration
	timer          *time.Timer
}

func newBackoffTimer(min, max time.Duration) *backoffTimer {
	return &backoffTimer{
		min:   min,
		max:   max,
		timer: time.NewTimer(min),
	}
}

func (t *backoffTimer) Stop() bool {
	return t.timer.Stop()
}

func (t *backoffTimer) channel() <-chan time.Time {
	return t.timer.C
}

func (t *backoffTimer) reset() bool {
	t.curr = t.min
	return t.timer.Reset(t.curr)
}

func (t *backoffTimer) backoff() bool {
	curr := t.curr * 2
	if curr > t.max {
		curr = t.max
	}
	t.curr = curr
	return t.timer.Reset(curr)
}
