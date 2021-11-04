package upstream

import (
	"reflect"
	"testing"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
)

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
			name:   "Returns repo ref string",
			fields: fields{
				git: &v1.Git{
					Repo:      "https://hostname/repo.git",
					Ref:       "main",
				},
			},
			want:   "https://hostname/repo.git@main",
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
			name:   "Sets upstream to git",
			fields: fields{
				git: &v1.Git{
					Repo:      "https://hostname/repo.git",
					Directory: "dir/path",
					Ref:       "main",
				},
			},
			args:   args{
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
