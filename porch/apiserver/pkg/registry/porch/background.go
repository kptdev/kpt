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
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/cache"
)

func RunBackground(coreClient client.WithWatch, cache *cache.Cache, stopCh <-chan struct{}) {
	b := background{
		coreClient: coreClient,
		cache:      cache,
	}
	ctx := context.Background()
	go b.run(ctx, stopCh)
}

// background manages background tasks
type background struct {
	coreClient client.WithWatch
	cache      *cache.Cache
}

// run will run until ctx is done or stopCh is closed
func (b *background) run(ctx context.Context, stopCh <-chan struct{}) {
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

	// Start ticker
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

loop:
	for {
		if watcher == nil {
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
				klog.Errorf("Cannot start watch: %v", err)
			} else {
				events = watcher.ResultChan()
			}
		}

		select {
		case event, eventOk := <-events:
			if !eventOk {
				klog.Errorf("Watch event stream closed. Will restart watch from bookmark %q", bookmark)
				watcher.Stop()
				events = nil
				watcher = nil
			} else if repository, ok := event.Object.(*configapi.Repository); ok {
				if event.Type == watch.Bookmark {
					bookmark = repository.ResourceVersion
					klog.Infof("Bookmark: %q", bookmark)
				} else {
					b.updateCache(ctx, event.Type, repository)
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

		case <-stopCh:
			klog.Info("Background routine exiting; stop channel closed")
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
		return b.cache.CloseRepository(repository)
	default:
		klog.Warning("Unhandled watch event type: %s", event)
	}
	return nil
}

func (b *background) runOnce(ctx context.Context) error {
	klog.Infof("background-refreshing repositories")
	var repositories configapi.RepositoryList
	if err := b.coreClient.List(ctx, &repositories); err != nil {
		return fmt.Errorf("error listing repository objects: %w", err)
	}

	for i := range repositories.Items {
		b.cacheRepository(ctx, &repositories.Items[i])
	}

	return nil
}

func (b *background) cacheRepository(ctx context.Context, repo *configapi.Repository) error {
	if _, err := b.cache.OpenRepository(repo); err != nil {
		return fmt.Errorf("error opening repository: %w", err)
	}
	return nil
}
