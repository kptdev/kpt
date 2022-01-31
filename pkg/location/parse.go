package location

import (
	"errors"
	"strings"
)

func ParseReference(location string, opts ...Option) (Reference, error) {
	opt := makeOptions(opts...)

	for _, parser := range opt.parsers {
		ref, err := parser(location, opt)
		if err != nil {
			return nil, err
		}
		if ref != nil {
			return ref, nil
		}
	}

	return nil, errors.New("not implemented")
}

func startsWith(value string, prefix string) (string, bool) {
	if parts := strings.SplitN(value, prefix, 2); len(parts) == 2 && len(parts[0]) == 0 {
		return parts[1], true
	}
	return prefix, false
}
