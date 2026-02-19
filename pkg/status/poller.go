// Copyright 2021 The kpt Authors
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

package status

import (
	"time"

	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/clusterreader"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/engine"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/statusreaders"
	"sigs.k8s.io/cli-utils/pkg/kstatus/watcher"
)

func NewStatusPoller(f util.Factory) (*polling.StatusPoller, error) {
	mapper, err := f.ToRESTMapper()
	if err != nil {
		return nil, err
	}

	return polling.NewStatusPollerFromFactory(f, polling.Options{
		CustomStatusReaders: []engine.StatusReader{
			&ConfigConnectorStatusReader{
				Mapper: mapper,
			},
			&RolloutStatusReader{
				Mapper: mapper,
			},
		},
	})
}

func NewStatusWatcher(f util.Factory) (watcher.StatusWatcher, error) {
	mapper, err := f.ToRESTMapper()
	if err != nil {
		return nil, err
	}

	dynamicClient, err := f.DynamicClient()
	if err != nil {
		return nil, err
	}

	return &watcher.DefaultStatusWatcher{
		DynamicClient: dynamicClient,
		Mapper:        mapper,
		ResyncPeriod:  1 * time.Hour,
		StatusReader: statusreaders.NewStatusReader(
			mapper,
			NewConfigConnectorStatusReader(mapper),
			NewRolloutStatusReader(mapper)),
		ClusterReader: &clusterreader.DynamicClusterReader{
			DynamicClient: dynamicClient,
			Mapper:        mapper,
		},
	}, nil
}
