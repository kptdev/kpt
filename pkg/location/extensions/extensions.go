package extensions

import (
	"context"
	"fmt"
)

type Reference interface {
	fmt.Stringer
	Type() string
	Validate() error
}

type ReferenceLock interface {
	Reference
}

type IdentifierGetter interface {
	GetIdentifier() (string, bool)
}

type LockGetter interface {
	GetLock() (string, bool)
}

// DefaultDirectoryNameGetter is present on Reference types that
// suggest a default local folder name
type DefaultDirectoryNameGetter interface {
	// GetDefaultDirectoryName implements the location.DefaultDirectoryName() method
	GetDefaultDirectoryName() (string, bool)
}

// DefaultIdentifierGetter is present on Reference types that
// suggest a default Identifier.
type DefaultIdentifierGetter interface {
	GetDefaultIdentifier(ctx context.Context) (string, error)
}

// RelPather is present on Reference types that
// will return a relative path if one reference is a sub-package
// location in another. The comparison is strict, meaning all criteria
// other than the directory component (like repo, ref, image, tag, etc.) must be equal.
type RelPather interface {
	Rel(targref Reference) (string, error)
}
