package mutate

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/location"
)

type IdentifierSetter interface {
	SetIdentifier(identifier string) (location.Reference, error)
}

func Identifier(ref location.Reference, identifier string) (location.Reference, error) {
	if ref, ok := ref.(IdentifierSetter); ok {
		return ref.SetIdentifier(identifier)
	}
	return nil, fmt.Errorf("changing identifier not supported for reference: %v", ref)
}

type LockSetter interface {
	SetLock(hash string) (location.ReferenceLock, error)
}

func Lock(ref location.Reference, hash string) (location.ReferenceLock, error) {
	if ref, ok := ref.(LockSetter); ok {
		return ref.SetLock(hash)
	}
	return nil, fmt.Errorf("locked reference not support for reference: %v", ref)
}
