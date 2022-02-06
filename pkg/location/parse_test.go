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
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
)

var (
	testReader  = &bytes.Buffer{}
	testWriter  = &bytes.Buffer{}
	testParsers = WithParsers(StdioParser, GitParser, OciParser, DirParser)
)

//nolint:scopelint
func TestParseReference(t *testing.T) {
	type args struct {
		location string
		opts     Option
	}
	type want struct {
		location  string
		reference Reference
	}
	tests := []struct {
		name    string
		args    args
		want    want
		wantErr bool
	}{
		{
			name: "OciSimpleName",
			args: args{
				location: "oci://ubuntu",
			},
			want: want{
				reference: Oci{
					Image:     name.MustParseReference("ubuntu"),
					Directory: ".",
				},
				location: "oci://ubuntu",
			},
			wantErr: false,
		},
		{
			name: "OciNameWithTag",
			args: args{
				location: "oci://my-registry.local/name:tag",
			},
			want: want{
				reference: Oci{
					Image:     name.MustParseReference("my-registry.local/name:tag"),
					Directory: ".",
				},
				location: "oci://my-registry.local/name:tag",
			},
			wantErr: false,
		},
		{
			name: "OciNameWithDigest",
			args: args{
				location: "oci://my-registry.local/name@sha256:9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba",
			},
			want: want{
				reference: Oci{
					Image:     name.MustParseReference("my-registry.local/name@sha256:9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba"),
					Directory: ".",
				},
				location: "oci://my-registry.local/name@sha256:9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba",
			},
			wantErr: false,
		},
		{
			name: "OciNameWithDirectory",
			args: args{
				location: "oci://my-registry.local/name//sub/directory",
			},
			want: want{
				reference: Oci{
					Image:     name.MustParseReference("my-registry.local/name:latest"),
					Directory: "sub/directory",
				},
				location: "oci://my-registry.local/name//sub/directory",
			},
			wantErr: false,
		},
		{
			name: "OciNameWithDirectoryAndTag",
			args: args{
				location: "oci://my-registry.local/name//sub/directory:tag",
			},
			want: want{
				reference: Oci{
					Image:     name.MustParseReference("my-registry.local/name:tag"),
					Directory: "sub/directory",
				},
				location: "oci://my-registry.local/name//sub/directory:tag",
			},
			wantErr: false,
		}, {
			name: "OciNameWithDirectoryAndDigest",
			args: args{
				location: "oci://my-registry.local/name//sub/directory@sha256:9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba",
			},
			want: want{
				reference: Oci{
					Image:     name.MustParseReference("my-registry.local/name@sha256:9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba"),
					Directory: "sub/directory",
				},
				location: "oci://my-registry.local/name//sub/directory@sha256:9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba",
			},
			wantErr: false,
		},
		{
			name: "OciDirectoryExtraSlashes",
			args: args{
				location: "oci://my-registry.local/name///sub/directory",
			},
			want: want{
				reference: Oci{
					Image:     name.MustParseReference("my-registry.local/name:latest"),
					Directory: "sub/directory",
				},
				location: "oci://my-registry.local/name///sub/directory",
			},
			wantErr: false,
		},
		{
			name: "OciDirectoryEmptyPath",
			args: args{
				location: "oci://my-registry.local/name//",
			},
			want: want{
				reference: Oci{
					Image:     name.MustParseReference("my-registry.local/name:latest"),
					Directory: ".",
				},
				location: "oci://my-registry.local/name//",
			},
			wantErr: false,
		},
		{
			name: "GitSimpleRepo",
			args: args{
				location: "https://hostname/repo.git@main",
			},
			want: want{
				reference: Git{
					Repo:      "https://hostname/repo",
					Directory: ".",
					Ref:       "main",
				},
				location: "https://hostname/repo.git@main",
			},
			wantErr: false,
		},
		{
			name: "GitWithDirectory",
			args: args{
				location: "https://hostname/repo.git/sub/directory@main",
			},
			want: want{
				reference: Git{
					Repo:      "https://hostname/repo",
					Directory: "sub/directory",
					Ref:       "main",
				},
				location: "https://hostname/repo.git/sub/directory@main",
			},
			wantErr: false,
		},
		{
			name: "SimpleDir",
			args: args{
				location: "path/to/directory",
			},
			want: want{
				reference: Dir{
					Directory: "path/to/directory",
				},
				location: "path/to/directory",
			},
			wantErr: false,
		},
		{
			name: "InputStream",
			args: args{
				location: "-",
				opts:     WithStdin(testReader),
			},
			want: want{
				reference: InputStream{
					Reader: testReader,
				},
				location: "-",
			},
			wantErr: false,
		},
		{
			name: "OutputStream",
			args: args{
				location: "-",
				opts:     WithStdout(testWriter),
			},
			want: want{
				reference: OutputStream{
					Writer: testWriter,
				},
				location: "-",
			},
			wantErr: false,
		},
		{
			name: "DuplexStream",
			args: args{
				location: "-",
				opts:     Options(WithStdin(testReader), WithStdout(testWriter)),
			},
			want: want{
				reference: DuplexStream{
					InputStream:  InputStream{Reader: testReader},
					OutputStream: OutputStream{Writer: testWriter},
				},
				location: "-",
			},
			wantErr: false,
		},
		{
			name: "StreamLocationNotExpected",
			args: args{
				location: "-",
			},
			want: want{
				reference: Dir{
					Directory: "-",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReference(tt.args.location, tt.args.opts, testParsers)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want.location != "" {
				if got.String() != tt.want.location {
					t.Errorf("Parse().String() = %v, want.location %v", got, tt.want.location)
				}
			}
			gotJSON, err := json.Marshal(got)
			if err != nil {
				t.Error(err)
			}
			wantJSON, err := json.Marshal(tt.want.reference)
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(gotJSON, wantJSON) {
				t.Errorf("Parse() = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}
