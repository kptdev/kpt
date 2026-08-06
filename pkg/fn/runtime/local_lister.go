package runtime

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

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
		return nil, fmt.Errorf("failed to list local tags using %q: %w; stderr: %s", l.Binary, err, stderr.String())
	}

	return linesToSlice(stdout.String()), nil
}

func linesToSlice(in string) []string {
	in = strings.TrimSpace(in)
	in = strings.ReplaceAll(in, "\r\n", "\n")
	var out []string
	for line := range strings.Lines(in) {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}

	return out
}
