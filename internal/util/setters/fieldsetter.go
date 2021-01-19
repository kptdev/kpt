package setters

import (
	"os"

	"github.com/GoogleContainerTools/kpt/internal/util/openapi"
	"github.com/go-errors/errors"
	"github.com/go-openapi/spec"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

// FieldSetter sets the value for a field setter.
type FieldSetter struct {
	// Name is the name of the setter to set
	Name string

	// Value is the value to set
	Value string

	// ListValues contains a list of values to set on a Sequence
	ListValues []string

	Description string

	SetBy string

	Count int

	OpenAPIPath string

	OpenAPIFileName string

	ResourcesPath string

	RecurseSubPackages bool

	IsSet bool

	SettersSchema *spec.Schema
}

// Set updates the OpenAPI definitions and resources with the new setter value
func (fs FieldSetter) Set() (int, error) {
	// the input field value is updated in the openAPI file and then parsed
	// at to get the value and set it to resource files, but if there is error
	// after updating openAPI file and while updating resources, the openAPI
	// file should be reverted, as set operation failed
	_, err := os.Stat(fs.OpenAPIPath)
	if err != nil {
		return 0, err
	}

	// Load the setter definitions
	sc, err := openapi.SchemaFromFile(fs.OpenAPIPath)
	if err != nil {
		return 0, err
	}
	if sc == nil {
		return 0, nil
	}

	fs.SettersSchema = sc
	if _, ok := sc.Definitions[fs.Name]; !ok {
		return 0, errors.Errorf("setter %q is not found", fs.Name)
	}

	// Update the resources with the new value
	// Set NoDeleteFiles to true as SetAll will return only the nodes of files which should be updated and
	// hence, rest of the files should not be deleted
	inout := &kio.LocalPackageReadWriter{PackagePath: fs.ResourcesPath, NoDeleteFiles: true, PackageFileName: fs.OpenAPIFileName}
	s := &Set{Name: fs.Name, SettersSchema: sc}
	err = kio.Pipeline{
		Inputs:  []kio.Reader{inout},
		Filters: []kio.Filter{SetAll(s)},
		Outputs: []kio.Writer{inout},
	}.Execute()

	return s.Count, err
}
