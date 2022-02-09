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

package location

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/location/extensions"
	"github.com/google/go-containerregistry/pkg/name"
)

//nolint:scopelint
func TestWithRevision(t *testing.T) {
	type args struct {
		ref      Reference
		revision string
	}
	tests := []struct {
		name    string
		args    args
		want    Reference
		wantErr bool
	}{
		{
			name: "OciWithRevision",
			args: args{
				ref: Oci{
					Image:     name.MustParseReference("my-registry.io/name:original"),
					Directory: "sub/directory",
				},
				revision: "updated",
			},
			want: Oci{
				Image:     name.MustParseReference("my-registry.io/name:updated"),
				Directory: "sub/directory",
			},
			wantErr: false,
		},
		{
			name: "GitWithRevision",
			args: args{
				ref: Git{
					Repo:      "repo",
					Directory: "sub/directory",
					Ref:       "original",
				},
				revision: "updated",
			},
			want: Git{
				Repo:      "repo",
				Directory: "sub/directory",
				Ref:       "updated",
			},
			wantErr: false,
		},
		{
			name: "CustomWithRevision",
			args: args{
				ref: custom{
					Place: "place",
					Label: "label",
				},
				revision: "new-label",
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
			got, err := WithRevision(tt.args.ref, tt.args.revision)
			if (err != nil) != tt.wantErr {
				t.Errorf("WithRevision() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithRevision() = %v, want %v", got, tt.want)
			}
		})
	}
}

type custom struct {
	Place string
	Label string
}

var _ extensions.Revisable = custom{}

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

func (ref custom) GetRevision() (string, bool) {
	return ref.Label, true
}
func (ref custom) WithRevision(revision string) (Reference, error) {
	return custom{
		Place: ref.Place,
		Label: revision,
	}, nil
}

func (ref custom) SetLock(lock string) (ReferenceLock, error) {
	return customLock{
		custom: ref,
		Lock:   lock,
	}, nil
}
