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

package rootsyncset

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/controllers/rootsyncsets/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

	client dynamic.Interface

	channel chan event.GenericEvent

	liens map[types.NamespacedName]struct{}
}

func (w watcher) watch() {
	clusterRefName := fmt.Sprintf("%s:%s", w.clusterRef.Kind, w.clusterRef.Name)
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
			klog.Infof("Starting watch for %s... ", clusterRefName)
			watcher, err = w.client.Resource(rootSyncGVR).Watch(w.ctx, v1.ListOptions{})
			if err != nil {
				klog.Errorf("Cannot start watch for %s: %v; will retry", clusterRefName, err)
				reconnect.backoff()
			} else {
				klog.Infof("Watch successfully started for %s.", clusterRefName)
				events = watcher.ResultChan()
			}
		case e, ok := <-events:
			if !ok {
				klog.Errorf("Watch event stream closed for cluster %s. Will restart watch from bookmark %q", clusterRefName, bookmark)
				watcher.Stop()
				events = nil
				watcher = nil

				// Initiate reconnect
				reconnect.reset()
			} else if obj, ok := e.Object.(*unstructured.Unstructured); ok {
				if e.Type == watch.Bookmark {
					bookmark = obj.GetResourceVersion()
					klog.Infof("Watch bookmark for %s: %q", clusterRefName, bookmark)
				} else {
					w.channel <- event.GenericEvent{
						Object: obj,
					}
				}
			} else {
				klog.V(5).Infof("Received unexpected watch event Object from %s: %T", e.Object, clusterRefName)
			}
		case <-w.ctx.Done():
			if w.ctx.Err() != nil {
				klog.V(2).Infof("exiting watcher for %s, because context is done: %v", clusterRefName, w.ctx.Err())
			} else {
				klog.Infof("Watch background routine exiting for %s; context done", clusterRefName)
			}
			break loop
		}
	}
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
