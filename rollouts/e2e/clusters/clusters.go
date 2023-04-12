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

package clusters

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CleanupBehavior string

// string mapping
const (
	CleanupDetach CleanupBehavior = "detach"
	CleanupDelete CleanupBehavior = "delete"
)

type ClusterSetup interface {
	// Create a cluster
	Add(name string, labels map[string]string) error
	// Wait for all clusters to become ready
	PrepareAndWait(ctx context.Context, timeout time.Duration) error
	// Cleanup deletes all clusters
	Cleanup(ctx context.Context) error
	// Get Cluster Reference
	GetClusterRefs() []map[string]interface{}
}

type Config struct {
	Count  int
	Prefix string
	Labels map[string]string
}

func GetClusterSetup(t *testing.T, c client.Client, cfg ...Config) (ClusterSetup, error) {
	return NewKindSetup(t, c, cfg...)
}
