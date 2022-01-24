// Copyright 2021 Google LLC
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
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/engine"
)

func NewStatusPoller(f util.Factory) (*polling.StatusPoller, error) {
	mapper, err := f.ToRESTMapper()
	if err != nil {
		return nil, err
	}

	return polling.NewStatusPollerFromFactory(f, []engine.StatusReader{
		&ConfigConnectorStatusReader{
			Mapper: mapper,
		},
	})
}
