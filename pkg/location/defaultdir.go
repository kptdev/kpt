package location

// DirectoryNameDefaulter is present on Reference types that
// suggest a default local folder name
type DirectoryNameDefaulter interface {
	// GetDefaultDirectoryName implements the location.DefaultDirectoryName() method
	GetDefaultDirectoryName() (string, bool)
}

// DefaultDirectoryName returns the suggested local directory name to
// create when a package from a remove reference is cloned or pulled.
// Returns an empty string and false if the Reference type does not have
// anything path-like to suggest from.
func DefaultDirectoryName(ref Reference) (string, bool) {
	if ref, ok := ref.(DirectoryNameDefaulter); ok {
		return ref.GetDefaultDirectoryName()
	}
	return "", false
}
