package setters

import (
	"io/ioutil"
	"os"

	"github.com/go-openapi/spec"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
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

func (fs *FieldSetter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	fs.Count, _ = fs.Set()
	return nil, nil
}

// Set updates the OpenAPI definitions and resources with the new setter value
func (fs FieldSetter) Set() (int, error) {
	// Update the OpenAPI definitions
	soa := SetOpenAPI{
		Name:        fs.Name,
		Value:       fs.Value,
		ListValues:  fs.ListValues,
		Description: fs.Description,
		SetBy:       fs.SetBy,
		IsSet:       fs.IsSet,
	}

	// the input field value is updated in the openAPI file and then parsed
	// at to get the value and set it to resource files, but if there is error
	// after updating openAPI file and while updating resources, the openAPI
	// file should be reverted, as set operation failed
	stat, err := os.Stat(fs.OpenAPIPath)
	if err != nil {
		return 0, err
	}

	curOpenAPI, err := ioutil.ReadFile(fs.OpenAPIPath)
	if err != nil {
		return 0, err
	}

	// write the new input value to openAPI file
	if err := soa.UpdateFile(fs.OpenAPIPath); err != nil {
		return 0, err
	}

	// Load the updated definitions
	sc, err := openapi.SchemaFromFile(fs.OpenAPIPath)
	if err != nil {
		return 0, err
	}
	fs.SettersSchema = sc

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

	// revert openAPI file if set operation fails
	if err != nil {
		if writeErr := ioutil.WriteFile(fs.OpenAPIPath, curOpenAPI, stat.Mode().Perm()); writeErr != nil {
			return 0, writeErr
		}
	}
	return s.Count, err
}
