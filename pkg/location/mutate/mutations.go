package mutate

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/location"
)

// IdentifierSetter is implemented by location.Reference types that
// support mutate.Identifier
type IdentifierSetter interface {
	// SetIdentifier is called by mutate.Identifier
	SetIdentifier(identifier string) (location.Reference, error)
}

// Identifier returns a new Reference where the property that
// identifies the branch, tag, or label has been replaced with value given.
// Typical identifier values are often a semantic name like 'draft', 'main', 'prod', or a
// string representation of a version. The specifics of how the identifier is
// mapped to storage depends on the type of reference.
func Identifier(ref location.Reference, identifier string) (location.Reference, error) {
	if ref, ok := ref.(IdentifierSetter); ok {
		return ref.SetIdentifier(identifier)
	}
	return nil, fmt.Errorf("changing identifier not supported for reference: %v", ref)
}

// LockSetter is implemented by location.Reference types that
// support mutate.Log
type LockSetter interface {
	SetLock(hash string) (location.ReferenceLock, error)
}

// Lock returns a new ReferenceLock where the property that identifies the
// unique commit or digest has been replaced with the value given.
// The exact meaning of the value depends on the type of reference, and
// is typically returned from the remote storage system as part of sending or
// receiving content.
func Lock(ref location.Reference, hash string) (location.ReferenceLock, error) {
	if ref, ok := ref.(LockSetter); ok {
		return ref.SetLock(hash)
	}
	return nil, fmt.Errorf("locked reference not support for reference: %v", ref)
}
