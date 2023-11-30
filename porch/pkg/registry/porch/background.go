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

package porch

import (
	"context"
	"fmt"
	"time"

	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/cache"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RunBackground(ctx context.Context, coreClient client.WithWatch, cache *cache.Cache) {
	b := background{
		coreClient: coreClient,
		cache:      cache,
	}
	go b.run(ctx)
}

// background manages background tasks
type background struct {
	coreClient client.WithWatch
	cache      *cache.Cache
}

const (
	minReconnectDelay = 1 * time.Second
	maxReconnectDelay = 30 * time.Second
)

// run will run until ctx is done
func (b *background) run(ctx context.Context) {
	klog.Infof("Background routine starting ...")

	// Repository watch.
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

	// Start ticker
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-reconnect.channel():
			var err error
			klog.Infof("Starting watch ... ")
			var obj configapi.RepositoryList
			watcher, err = b.coreClient.Watch(ctx, &obj, &client.ListOptions{
				Raw: &v1.ListOptions{
					AllowWatchBookmarks: true,
					ResourceVersion:     bookmark,
				},
			})
			if err != nil {
				klog.Errorf("Cannot start watch: %v; will retry", err)
				reconnect.backoff()
			} else {
				klog.Infof("Watch successfully started.")
				events = watcher.ResultChan()
			}

		case event, eventOk := <-events:
			if !eventOk {
				klog.Errorf("Watch event stream closed. Will restart watch from bookmark %q", bookmark)
				watcher.Stop()
				events = nil
				watcher = nil

				// Initiate reconnect
				reconnect.reset()
			} else if repository, ok := event.Object.(*configapi.Repository); ok {
				if event.Type == watch.Bookmark {
					bookmark = repository.ResourceVersion
					klog.Infof("Bookmark: %q", bookmark)
				} else {
					if err := b.updateCache(ctx, event.Type, repository); err != nil {
						klog.Warningf("error updating cache: %v", err)
					}
				}
			} else {
				klog.V(5).Infof("Received unexpected watch event Object: %T", event.Object)
			}

		case t := <-ticker.C:
			klog.Infof("Background task %s", t)
			if err := b.runOnce(ctx); err != nil {
				klog.Errorf("Periodic repository refresh failed: %v", err)
			}

		case <-ctx.Done():
			if ctx.Err() != nil {
				klog.V(2).Infof("exiting background poller, because context is done: %v", ctx.Err())
			} else {
				klog.Infof("Background routine exiting; context done")
			}
			break loop
		}
	}
}

func (b *background) updateCache(ctx context.Context, event watch.EventType, repository *configapi.Repository) error {
	switch event {
	case watch.Added:
		klog.Infof("Repository added: %s:%s", repository.ObjectMeta.Namespace, repository.ObjectMeta.Name)
		return b.cacheRepository(ctx, repository)
	case watch.Modified:
		klog.Infof("Repository modified: %s:%s", repository.ObjectMeta.Namespace, repository.ObjectMeta.Name)
		// TODO: implement
	case watch.Deleted:
		klog.Infof("Repository deleted: %s:%s", repository.ObjectMeta.Namespace, repository.ObjectMeta.Name)
		shared, err := b.isSharedRepository(ctx, repository)
		if err != nil {
			return err
		}
		// Only close the repository if no other k8s repository resources references
		// the same underlying git/oci repo.
		if !shared {
			return b.cache.CloseRepository(repository)
		}
		return nil
	default:
		klog.Warning("Unhandled watch event type: %s", event)
	}
	return nil
}

// isSharedRepository checks if the underlying git/oci repo of the provided
// k8s repository is also used by another repository.
func (b *background) isSharedRepository(ctx context.Context, repo *configapi.Repository) (bool, error) {
	var obj configapi.RepositoryList
	if err := b.coreClient.List(ctx, &obj); err != nil {
		return false, err
	}
	for _, r := range obj.Items {
		if r.Name == repo.Name && r.Namespace == repo.Namespace {
			continue
		}
		if r.Spec.Type != repo.Spec.Type {
			continue
		}
		switch r.Spec.Type {
		case configapi.RepositoryTypeOCI:
			if r.Spec.Oci.Registry == repo.Spec.Oci.Registry {
				return true, nil
			}
		case configapi.RepositoryTypeGit:
			if r.Spec.Git.Repo == repo.Spec.Git.Repo && r.Spec.Git.Directory == repo.Spec.Git.Directory {
				return true, nil
			}
		default:
			return false, fmt.Errorf("type %q not supported", r.Spec.Type)
		}
	}
	return false, nil
}

func (b *background) runOnce(ctx context.Context) error {
	klog.Infof("background-refreshing repositories")
	var repositories configapi.RepositoryList
	if err := b.coreClient.List(ctx, &repositories); err != nil {
		return fmt.Errorf("error listing repository objects: %w", err)
	}

	for i := range repositories.Items {
		repo := &repositories.Items[i]

		if err := b.cacheRepository(ctx, repo); err != nil {
			klog.Errorf("Failed to cache repository: %v", err)
		}
	}

	return nil
}

func (b *background) cacheRepository(ctx context.Context, repo *configapi.Repository) error {
	var condition v1.Condition
	if _, err := b.cache.OpenRepository(ctx, repo); err == nil {
		condition = v1.Condition{
			Type:               configapi.RepositoryReady,
			Status:             v1.ConditionTrue,
			ObservedGeneration: repo.Generation,
			LastTransitionTime: v1.Now(),
			Reason:             configapi.ReasonReady,
		}
	} else {
		condition = v1.Condition{
			Type:               configapi.RepositoryReady,
			Status:             v1.ConditionFalse,
			ObservedGeneration: repo.Generation,
			LastTransitionTime: v1.Now(),
			Reason:             configapi.ReasonError,
			Message:            err.Error(),
		}
	}

	meta.SetStatusCondition(&repo.Status.Conditions, condition)
	if err := b.coreClient.Status().Update(ctx, repo); err != nil {
		return fmt.Errorf("error updating repository status: %w", err)
	}
	return nil
}

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
