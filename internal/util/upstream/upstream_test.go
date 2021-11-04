package upstream

import (
	"reflect"
	"testing"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
)

func TestNewUpstream(t *testing.T) {
	type args struct {
		kf *kptfilev1.KptFile
	}
	tests := []struct {
		name    string
		args    args
		want    Fetcher
		wantErr bool
	}{
		{
			name:    "returns git upstream",
			args:    args{
				kf: &kptfilev1.KptFile{
					Upstream:     &kptfilev1.Upstream{
						Type:           kptfilev1.GitOrigin,
						Git:            &kptfilev1.Git{
							Repo:      "repo-name",
							Directory: "dir-name",
							Ref:       "ref-name",
						},						
					},
				},
			},
			want:    &gitUpstream{
				git: &kptfilev1.Git{
					Repo:      "repo-name",
					Directory: "dir-name",
					Ref:       "ref-name",
				},
			},
			wantErr: false,
		},
		{
			name:    "returns oci upstream",
			args:    args{
				kf: &kptfilev1.KptFile{
					Upstream:     &kptfilev1.Upstream{
						Type:           kptfilev1.OciOrigin,
						Oci:            &kptfilev1.Oci{
							Image: "image-name",
						},						
					},
				},
			},
			want:    &ociUpstream{
				image: "image-name",
			},
			wantErr: false,
		},
		{
			name:    "empty type fails",
			args:    args{
				kf: &kptfilev1.KptFile{
					Upstream:     &kptfilev1.Upstream{
					},
				},
			},
			want: nil,
			wantErr: true,
		},
		{
			name:    "nil upstream fails",
			args:    args{
				kf: &kptfilev1.KptFile{},
			},
			want: nil,
			wantErr: true,
		},
		{
			name:    "nil git fails",
			args:    args{
				kf: &kptfilev1.KptFile{
					Upstream:     &kptfilev1.Upstream{
						Type:           kptfilev1.GitOrigin,
					},
				},
			},
			want: nil,
			wantErr: true,
		},
		{
			name:    "nil oci fails",
			args:    args{
				kf: &kptfilev1.KptFile{
					Upstream:     &kptfilev1.Upstream{
						Type:           kptfilev1.OciOrigin,
					},
				},
			},
			want: nil,
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
