package fnruntime

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/regclient/regclient"
	regclientref "github.com/regclient/regclient/types/ref"
)

// RegClientLister is a TagLister using the regclient module to list remote OCI tags.
type RegClientLister struct {
	client *regclient.RegClient
}

var _ TagLister = &RegClientLister{}

func (l *RegClientLister) Name() string {
	return "regclient"
}

func (l *RegClientLister) List(ctx context.Context, image string) ([]string, error) {
	ref, err := regclientref.New(image)
	if err != nil {
		return nil, err
	}

	defer func() { _ = l.client.Close(ctx, ref) }()

	tagList, err := l.client.TagList(ctx, ref)
	if err != nil {
		return nil, err
	}

	return tagList.GetTags()
}

// LocalLister is a TagLister using the given CLI tool to list local OCI tags
type LocalLister struct {
	Binary string
}

var _ TagLister = &LocalLister{}

func (l *LocalLister) Name() string {
	return "local-" + l.Binary
}

func (l *LocalLister) List(ctx context.Context, image string) ([]string, error) {
	command := exec.CommandContext(ctx, l.Binary, "image", "ls", "--filter", "reference="+image, "--format", "{{ .Tag }}")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command.Stdout = stdout
	command.Stderr = stderr

	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("error whilst listing local tags using %q: %w; stderr: %s", l.Binary, err, stderr.String())
	}

	return linesToSlice(stdout.String()), nil
}

func linesToSlice(in string) []string {
	in = strings.TrimSpace(in)
	in = strings.ReplaceAll(in, "\r\n", "\n")
	var out []string
	for line := range strings.Lines(in) {
		line = strings.TrimSpace(line)
		out = append(out, line)
	}

	return out
}
