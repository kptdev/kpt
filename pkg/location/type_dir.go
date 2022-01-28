package location

import "fmt"

type Dir struct {
	Directory string
}

var _ Reference = Dir{}

func (ref Dir) String() string {
	return fmt.Sprintf("type:dir directory:%q", ref.Directory)
}
