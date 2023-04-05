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
	"testing"

	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
)

func TestApiserverProvider(t *testing.T) {

	uip := &ApiserverUserInfoProvider{}

	for _, tc := range []struct {
		name   string
		groups []string

		want *repository.UserInfo
	}{
		{
			want: nil,
		},
		{
			name:   "user1@domain.com",
			groups: []string{},
			want:   nil,
		},
		{
			name:   "user2@domain.com",
			groups: []string{user.AllUnauthenticated, user.Anonymous},
			want:   nil,
		},
		{
			name:   "",
			groups: []string{user.AllAuthenticated},
			want:   nil,
		},
		{
			name:   "user3@domain.com",
			groups: []string{user.AllAuthenticated},
			want: &repository.UserInfo{
				Name:  "user3@domain.com",
				Email: "user3@domain.com",
			},
		},
	} {
		di := &user.DefaultInfo{
			Name:   tc.name,
			UID:    "uuid",
			Groups: tc.groups,
			Extra:  map[string][]string{},
		}

		ctx := request.WithUser(context.Background(), di)

		if got, want := uip.GetUserInfo(ctx), tc.want; !cmp.Equal(got, want) {
			t.Errorf("GetUserInfo: got %v, want %v; diff (-want,+got): %s", got, want, cmp.Diff(want, got))
		}
	}
}

func TestEmptyUserInfo(t *testing.T) {
	uip := &ApiserverUserInfoProvider{}

	if got, want := uip.GetUserInfo(context.Background()), (*repository.UserInfo)(nil); got != want {
		t.Errorf("GetUserInfo with empty context: got %v, want %v", got, want)
	}
}
