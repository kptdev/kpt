package extensions

import "context"

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
