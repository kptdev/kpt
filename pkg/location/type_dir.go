package location

import (
	"fmt"
	"io/fs"
)

type Dir struct {
	Directory string
}

var _ Reference = Dir{}

func parseDir(location string, opt options) (Reference, error) {
	if fs.ValidPath(location) {
		return Dir{
			Directory: location,
		}, nil
	}
	return nil, nil
}

func (ref Dir) String() string {
	return fmt.Sprintf("type:dir directory:%q", ref.Directory)
}

func (ref Dir) Type() string {
	return "dir"
}

func (ref Dir) Validate() error {
	return nil
}
