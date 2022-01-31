package location

import "fmt"

type DirectoryNameDefaulter interface {
	GetDefaultDirectoryName() (string, error)
}

func DefaultDirectoryName(ref Reference) (string, error) {
	if ref, ok := ref.(DirectoryNameDefaulter); ok {
		return ref.GetDefaultDirectoryName()
	}
	return "", fmt.Errorf("default directory not supported for reference: %v", ref)
}
