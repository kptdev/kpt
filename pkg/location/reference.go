package location

import "fmt"

type Reference interface {
	fmt.Stringer
	Type() string
	Validate() error
}

type ReferenceLock interface {
	Reference
}
