package location

import (
	"errors"
	"path/filepath"
	"strings"
)

func ParseReference(location string, opts ...Option) (Reference, error) {
	opt := makeOptions(opts...)

	if location == "-" {
		switch {
		case opt.stdin != nil && opt.stdout != nil:
			return InputOutputStream{
				Reader: opt.stdin,
				Writer: opt.stdout,
			}, nil
		case opt.stdin != nil:
			return InputStream{
				Reader: opt.stdin,
			}, nil
		case opt.stdout != nil:
			return OutputStream{
				Writer: opt.stdout,
			}, nil
		}
		return nil, errors.New("stdin/stdout not supported here")
	}

	if _, ok := startsWith(location, "oci://"); ok {
		oci, ociErr := NewOci(location, opts...)
		return oci, ociErr
	}

	git, gitErr := NewGit(location, opts...)
	if gitErr == nil {
		return git, nil
	}

	if s, ok := isDir(location); ok {
		return Dir{
			Directory: s,
		}, nil
	}

	return nil, errors.New("not implemented")
}

func startsWith(value string, prefix string) (string, bool) {
	if parts := strings.SplitN(value, prefix, 2); len(parts) == 2 && len(parts[0]) == 0 {
		return parts[1], true
	}
	return prefix, false
}

func isDir(value string) (string, bool) {
	return filepath.Clean(value), true
}
