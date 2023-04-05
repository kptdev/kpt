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
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

type errNotAcceptable struct {
	resource schema.GroupResource
}

func (e errNotAcceptable) Error() string {
	return fmt.Sprintf("%s does not support Table format", e.resource)
}

func (e errNotAcceptable) Status() metav1.Status {
	return metav1.Status{
		Status:  metav1.StatusFailure,
		Message: e.Error(),
		Reason:  metav1.StatusReasonNotAcceptable,
		Code:    http.StatusNotAcceptable,
	}
}

func newResourceNotAcceptableError(ctx context.Context, resource schema.GroupResource) error {
	if info, ok := genericapirequest.RequestInfoFrom(ctx); ok {
		resource = schema.GroupResource{
			Group:    info.APIGroup,
			Resource: info.Resource,
		}
	}
	return errNotAcceptable{
		resource: resource,
	}
}
