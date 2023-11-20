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

package engine

import (
	"context"
	"sync"

	"github.com/GoogleContainerTools/kpt/porch/pkg/meta"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

// ObjectCache caches objects across repositories, and allows for watching.
type WatcherManager interface {
	WatchPackageRevisions(ctx context.Context, filter repository.ListPackageRevisionFilter, callback ObjectWatcher) error
}

// PackageRevisionWatcher is the callback interface for watchers.
type ObjectWatcher interface {
	OnPackageRevisionChange(eventType watch.EventType, obj repository.PackageRevision, objMeta meta.PackageRevisionMeta) bool
}

func NewWatcherManager() *watcherManager {
	return &watcherManager{}
}

// watcherManager implements WatcherManager
type watcherManager struct {
	mutex sync.Mutex

	// watchers is a list of all the change-listeners.
	// As an optimization, values in this slice can be nil; we use this when the watch ends.
	watchers []*watcher
}

// watcher is a single change listener.
type watcher struct {
	// isDoneFunction should return non-nil when the watcher is finished.
	// This is normally bound to ctx.Err()
	isDoneFunction func() error

	// callback is called for each object change.
	callback ObjectWatcher

	// filter can limit the objects reported.
	filter repository.ListPackageRevisionFilter
}

// WatchPackageRevision adds a change-listener that will be called for all changes.
func (r *watcherManager) WatchPackageRevisions(ctx context.Context, filter repository.ListPackageRevisionFilter, callback ObjectWatcher) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// reap any dead watchers
	for i, watcher := range r.watchers {
		if watcher == nil {
			continue
		}
		if err := watcher.isDoneFunction(); err != nil {
			klog.Infof("stopping watcher in reaper: %v", err)
			r.watchers[i] = nil
			continue
		}
	}

	w := &watcher{
		isDoneFunction: ctx.Err,
		callback:       callback,
		filter:         filter,
	}

	active := 0
	// See if we have an empty slot in the watchers list
	inserted := false
	for i, watcher := range r.watchers {
		if watcher != nil {
			active += 1
		} else if !inserted {
			active += 1
			r.watchers[i] = w
			inserted = true
		}
	}

	if !inserted {
		// We didn't slot it in to an existing slot, append it
		active += 1
		r.watchers = append(r.watchers, w)
	}

	klog.Infof("added watcher %p; there are now %d active watchers and %d slots", w, active, len(r.watchers))
	return nil
}

// notifyPackageRevisionChange is called to send a change notification to all interested listeners.
func (r *watcherManager) NotifyPackageRevisionChange(eventType watch.EventType, obj repository.PackageRevision, objMeta meta.PackageRevisionMeta) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	sent := 0
	for i, watcher := range r.watchers {
		if watcher == nil {
			continue
		}
		if err := watcher.isDoneFunction(); err != nil {
			klog.Infof("stopping watcher in response to error %v", err)
			r.watchers[i] = nil
			continue
		}
		if keepGoing := watcher.callback.OnPackageRevisionChange(eventType, obj, objMeta); !keepGoing {
			klog.Infof("stopping watcher in response to !keepGoing")
			r.watchers[i] = nil
		}
		sent += 1
	}

	return sent
}
