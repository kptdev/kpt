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

package upstream

import (
	"reflect"
	"testing"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
)

//nolint:scopelint
func TestNewGitFetcher(t *testing.T) {
	type args struct {
		git *v1.Git
	}
	tests := []struct {
		name string
		args args
		want Fetcher
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewGitUpstream(tt.args.git); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGitFetcher() = %v, want %v", got, tt.want)
			}
		})
	}
}

//nolint:scopelint
func Test_gitUpstream_String(t *testing.T) {
	type fields struct {
		git *v1.Git
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Returns repo ref string",
			fields: fields{
				git: &v1.Git{
					Repo: "https://hostname/repo.git",
					Ref:  "main",
				},
			},
			want: "https://hostname/repo.git@main",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &gitUpstream{
				git: tt.fields.git,
			}
			if got := u.String(); got != tt.want {
				t.Errorf("gitUpstream.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

//nolint:scopelint
func Test_gitUpstream_ApplyUpstream(t *testing.T) {
	type fields struct {
		git *v1.Git
	}
	type args struct {
		kf *v1.KptFile
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Sets upstream to git",
			fields: fields{
				git: &v1.Git{
					Repo:      "https://hostname/repo.git",
					Directory: "dir/path",
					Ref:       "main",
				},
			},
			args: args{
				kf: &v1.KptFile{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &gitUpstream{
				git: tt.fields.git,
			}
			u.ApplyUpstream(tt.args.kf)
		})
	}
}
