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

package remoteclient

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	cloudresourcemanagerv1 "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ProjectCache struct {
}

type ProjectInfo struct {
	ProjectID     string
	ProjectNumber int64
}

// Init performs one-off initialization of the object.
func (r *ProjectCache) Init(mgr ctrl.Manager) error {
	return nil
}

func (r *ProjectCache) LookupByProjectID(ctx context.Context, projectID string, tokenSource oauth2.TokenSource) (*ProjectInfo, error) {
	// TODO: Cache

	crmClient, err := cloudresourcemanagerv1.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create new cloudresourcemanager client: %w", err)
	}

	project, err := crmClient.Projects.Get(projectID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("error querying project %q: %w", projectID, err)
	}

	return &ProjectInfo{
		ProjectID:     project.ProjectId,
		ProjectNumber: project.ProjectNumber,
	}, nil
}
