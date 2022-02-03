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

package mutate

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/location"
	"github.com/google/go-containerregistry/pkg/name"
)

//nolint:scopelint
func TestSetIdentifier(t *testing.T) {
	type args struct {
		ref        location.Reference
		identifier string
	}
	tests := []struct {
		name    string
		args    args
		want    location.Reference
		wantErr bool
	}{
		{
			name: "OciWithIdentifier",
			args: args{
				ref: location.Oci{
					Image:     name.MustParseReference("my-registry.io/name:original"),
					Directory: "sub/directory",
				},
				identifier: "updated",
			},
			want: location.Oci{
				Image:     name.MustParseReference("my-registry.io/name:updated"),
				Directory: "sub/directory",
			},
			wantErr: false,
		},
		{
			name: "GitWithIdentifier",
			args: args{
				ref: location.Git{
					Repo:      "repo",
					Directory: "sub/directory",
					Ref:       "original",
				},
				identifier: "updated",
			},
			want: location.Git{
				Repo:      "repo",
				Directory: "sub/directory",
				Ref:       "updated",
			},
			wantErr: false,
		},
		{
			name: "CustomWithIdentifier",
			args: args{
				ref: custom{
					Place: "place",
					Label: "label",
				},
				identifier: "new-label",
			},
			want: custom{
				Place: "place",
				Label: "new-label",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Identifier(tt.args.ref, tt.args.identifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("WithIdentifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}

//nolint:scopelint
func TestSetLock(t *testing.T) {
	type args struct {
		ref  location.Reference
		lock string
	}
	tests := []struct {
		name    string
		args    args
		want    location.ReferenceLock
		wantErr bool
	}{
		{
			name: "OciWithLock",
			args: args{
				ref: location.Oci{
					Image:     name.MustParseReference("my-registry.io/name:original"),
					Directory: "sub/directory",
				},
				lock: "sha256:9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba",
			},
			want: location.OciLock{
				Oci: location.Oci{
					Image:     name.MustParseReference("my-registry.io/name:original"),
					Directory: "sub/directory",
				},
				Digest: name.MustParseReference("my-registry.io/name@sha256:9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba"),
			},
			wantErr: false,
		},
		{
			name: "GitWithLock",
			args: args{
				ref: location.Git{
					Repo:      "repo",
					Directory: "sub/directory",
					Ref:       "original",
				},
				lock: "9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba",
			},
			want: location.GitLock{
				Git: location.Git{
					Repo:      "repo",
					Directory: "sub/directory",
					Ref:       "original",
				},
				Commit: "9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba",
			},
			wantErr: false,
		},
		{
			name: "CustomWithLock",
			args: args{
				ref: custom{
					Place: "place",
					Label: "label",
				},
				lock: "lock",
			},
			want: customLock{
				custom: custom{
					Place: "place",
					Label: "label",
				},
				Lock: "lock",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Lock(tt.args.ref, tt.args.lock)
			if (err != nil) != tt.wantErr {
				t.Errorf("WithLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithLock() = %v, want %v", got, tt.want)
			}
		})
	}
}

type custom struct {
	Place string
	Label string
}

type customLock struct {
	custom
	Lock string
}

func (ref custom) String() string {
	return fmt.Sprintf("place:%s label:%s", ref.Place, ref.Label)
}

func (ref custom) Type() string {
	return "custom"
}

func (ref custom) Validate() error {
	return nil
}

func (ref custom) SetIdentifier(identifier string) (location.Reference, error) {
	return custom{
		Place: ref.Place,
		Label: identifier,
	}, nil
}

func (ref custom) SetLock(lock string) (location.ReferenceLock, error) {
	return customLock{
		custom: ref,
		Lock:   lock,
	}, nil
}
