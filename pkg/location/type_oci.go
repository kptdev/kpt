package location

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

type Oci struct {
	Image     name.Reference
	Directory string
}

var _ Reference = Oci{}

type OciLock struct {
	Oci
	Digest name.Reference
}

var _ Reference = OciLock{}
var _ ReferenceLock = OciLock{}

func NewOci(location string, opts ...Option) (Oci, error) {

	if s, ok := startsWith(location, "oci://"); ok {
		ref, err := name.ParseReference(s)
		if err != nil {
			return Oci{}, err
		}
		if parts := strings.SplitN(ref.Context().Name(), "//", 2); len(parts) == 2 {
			repo, err := name.NewRepository(parts[0])
			if err != nil {
				return Oci{}, err
			}

			dir := filepath.Clean(parts[1])
			if filepath.IsAbs(dir) {
				dir, err = filepath.Rel("/", dir)
				if err != nil {
					return Oci{}, err
				}
			}

			switch ref := ref.(type) {
			case name.Tag:
				return Oci{
					Image:     repo.Tag(ref.TagStr()),
					Directory: dir,
				}, nil
			case name.Digest:
				return Oci{
					Image:     repo.Digest(ref.DigestStr()),
					Directory: dir,
				}, nil
			}
		}
		return Oci{
			Image:     ref,
			Directory: ".",
		}, nil
	}

	return Oci{}, errors.New("invalid format")
}

func (ref Oci) String() string {
	return fmt.Sprintf("type:oci image:%q directory:%q", ref.Image, ref.Directory)
}

func (ref OciLock) String() string {
	return fmt.Sprintf("%v digest:%q", ref.Oci, ref.Digest)
}

func (ref Oci) SetIdentifier(identifier string) (Reference, error) {
	return Oci{
		Image:     ref.Image.Context().Tag(identifier),
		Directory: ref.Directory,
	}, nil
}

func (ref Oci) SetLock(lock string) (ReferenceLock, error) {
	return OciLock{
		Oci:    ref,
		Digest: ref.Image.Context().Digest(lock),
	}, nil
}
