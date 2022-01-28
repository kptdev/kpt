package mutate

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/location"
)

type IdentifierSetter interface {
	SetIdentifier(identifier string) (location.Reference, error)
}

func Identifier(ref location.Reference, identifier string) (location.Reference, error) {
	switch ref := ref.(type) {
	case IdentifierSetter:
		return ref.SetIdentifier(identifier)
	}
	return nil, fmt.Errorf("identifier not supported for reference: %v", ref)
}

type LockSetter interface {
	SetLock(hash string) (location.ReferenceLock, error)
}

func Lock(ref location.Reference, hash string) (location.ReferenceLock, error) {
	switch ref := ref.(type) {
	case LockSetter:
		return ref.SetLock(hash)
	}
	return nil, fmt.Errorf("locked reference not support for reference: %v", ref)
}
