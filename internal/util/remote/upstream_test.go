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

package remote

import (
	"reflect"
	"testing"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
)

//nolint:scopelint
func TestNewUpstream(t *testing.T) {
	type args struct {
		kf *kptfilev1.KptFile
	}
	tests := []struct {
		name    string
		args    args
		want    Upstream
		wantErr bool
	}{
		{
			name: "returns git upstream",
			args: args{
				kf: &kptfilev1.KptFile{
					Upstream: &kptfilev1.Upstream{
						Type: kptfilev1.GitOrigin,
						Git: &kptfilev1.Git{
							Repo:      "repo-name",
							Directory: "dir-name",
							Ref:       "ref-name",
						},
					},
				},
			},
			want: &gitUpstream{
				git: &kptfilev1.Git{
					Repo:      "repo-name",
					Directory: "dir-name",
					Ref:       "ref-name",
				},
				gitLock: &kptfilev1.GitLock{},
			},
			wantErr: false,
		},
		{
			name: "returns oci upstream",
			args: args{
				kf: &kptfilev1.KptFile{
					Upstream: &kptfilev1.Upstream{
						Type: kptfilev1.OciOrigin,
						Oci: &kptfilev1.Oci{
							Image: "image-name",
						},
					},
				},
			},
			want: &ociUpstream{
				oci: &kptfilev1.Oci{
					Image: "image-name",
				},
				ociLock: &kptfilev1.OciLock{},
			},
			wantErr: false,
		},
		{
			name: "empty type fails",
			args: args{
				kf: &kptfilev1.KptFile{
					Upstream: &kptfilev1.Upstream{},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nil upstream fails",
			args: args{
				kf: &kptfilev1.KptFile{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nil git fails",
			args: args{
				kf: &kptfilev1.KptFile{
					Upstream: &kptfilev1.Upstream{
						Type: kptfilev1.GitOrigin,
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nil oci fails",
			args: args{
				kf: &kptfilev1.KptFile{
					Upstream: &kptfilev1.Upstream{
						Type: kptfilev1.OciOrigin,
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewUpstream(tt.args.kf)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUpstream() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUpstream() = %v, want %v", got, tt.want)
			}
		})
	}
}
