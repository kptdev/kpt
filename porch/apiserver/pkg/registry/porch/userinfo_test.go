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
	"strings"
	"testing"

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
)

func TestApiserverProvider(t *testing.T) {

	uip := &ApiserverUserInfoProvider{}

	for _, tc := range []struct {
		name   string
		groups []string

		want_user string
		want_ok   bool
	}{
		{
			want_user: "",
			want_ok:   false,
		},
		{
			name:      "user1@domain.com",
			groups:    []string{},
			want_user: "",
			want_ok:   false,
		},
		{
			name:      "user2@domain.com",
			groups:    []string{user.AllUnauthenticated, user.Anonymous},
			want_user: "",
			want_ok:   false,
		},
		{
			name:      "",
			groups:    []string{user.AllAuthenticated},
			want_user: "",
			want_ok:   false,
		},
		{
			name:      "user3@domain.com",
			groups:    []string{user.AllAuthenticated},
			want_user: "user3@domain.com",
			want_ok:   true,
		},
	} {
		di := &user.DefaultInfo{
			Name:   tc.name,
			UID:    "uuid",
			Groups: tc.groups,
			Extra:  map[string][]string{},
		}

		ctx := request.WithUser(context.Background(), di)

		got_user, got_ok := uip.GetUserName(ctx)

		if got_ok != tc.want_ok {
			t.Errorf("GetUserName with user %q and groups: %s: got ok flag %t, want %t", tc.name, strings.Join(tc.groups, ","), got_ok, tc.want_ok)
		}

		if got_user != tc.want_user {
			t.Errorf("GetUserName with user %q and groups: %s: got user %q, want %q", tc.name, strings.Join(tc.groups, ","), got_user, tc.want_user)
		}
	}
}

func TestEmptyUserInfo(t *testing.T) {
	uip := &ApiserverUserInfoProvider{}

	got_user, got_ok := uip.GetUserName(context.Background())
	want_user, want_ok := "", false

	if got_ok != want_ok {
		t.Errorf("GetUserName with empty context: got ok %t, want %t", got_ok, want_ok)
	}

	if got_user != want_user {
		t.Errorf("GetUserName with empty context: got user %q, want %q", got_user, want_user)
	}
}
