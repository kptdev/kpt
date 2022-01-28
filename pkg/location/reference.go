package location

import "fmt"

type Reference interface {
	fmt.Stringer
}

type ReferenceLock interface {
	Reference
}
