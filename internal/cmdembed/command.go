// Copyright 2019 Google LLC
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

package cmdembed

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
)

const (
	ImageFnPath = "/localconfig/fn-config.yaml"
)

func NewRunner(ctx context.Context, parent string) *Runner {
	r := &Runner{
		Ctx: ctx,
	}
	c := &cobra.Command{
		Use:     "embed SOURCE_IMAGE TARGET_IMAGE --fn-config=fn-config.yaml",
		Args:    cobra.ExactArgs(2),
		RunE:    r.runE,
		PreRunE: r.preRunE,
	}
	r.Command = c
	r.Command.Flags().StringVar(
		&r.FnConfigPath, "fn-config", "", "path to the function config file")
	cmdutil.FixDocs("kpt", parent, c)
	return r
}

func NewCommand(ctx context.Context, parent string) *cobra.Command {
	return NewRunner(ctx, parent).Command
}

type Runner struct {
	Command *cobra.Command
	Ctx     context.Context

	SourceImage  string
	TargetImage  string
	FnConfigPath string
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	r.SourceImage = args[0]
	r.TargetImage = args[1]
	if r.FnConfigPath == "" {
		return fmt.Errorf("path to function config file must be provided")
	}
	return nil
}

func (r *Runner) runE(c *cobra.Command, _ []string) error {
	img, err := crane.Pull(r.SourceImage)
	if err != nil {
		return fmt.Errorf("unable to pull image %q: %w", r.SourceImage, err)
	}
	fmt.Fprintf(c.OutOrStdout(), "Pulled image %q\n", r.SourceImage)

	addLayer, err := layerFromFile(r.FnConfigPath)
	if err != nil {
		return fmt.Errorf("unable to create new layer with config: %w", err)
	}
	fmt.Fprintf(c.OutOrStdout(), "Created new layer with config %q\n", r.FnConfigPath)

	newImg, err := mutate.AppendLayers(img, addLayer)
	if err != nil {
		return fmt.Errorf("unable to add layer to image: %w", err)
	}

	tag, err := name.NewTag(r.TargetImage)
	if err != nil {
		return fmt.Errorf("unable to tag image %q: %w", r.TargetImage, err)
	}

	if _, err := daemon.Write(tag, newImg); err != nil {
		return fmt.Errorf("unable to write image %q: %w", r.TargetImage, err)
	}
	fmt.Fprintf(c.OutOrStdout(), "Created new image %q\n", r.TargetImage)
	return nil
}

func layerFromFile(path string) (v1.Layer, error) {
	var b bytes.Buffer
	tw := tar.NewWriter(&b)

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	hdr := &tar.Header{
		Name:     ImageFnPath,
		Mode:     int64(0644),
		Size:     info.Size(),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, fmt.Errorf("failed to write tar header: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := io.Copy(tw, f); err != nil {
		return nil, fmt.Errorf("failed to read file into the tar: %w", err)
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to finish tar: %w", err)
	}
	return tarball.LayerFromReader(&b)
}
