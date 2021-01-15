// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package setters

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/go-openapi/spec"
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fieldmeta"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/pathutil"
	"sigs.k8s.io/kustomize/kyaml/setters2"
	"sigs.k8s.io/kustomize/kyaml/setters2/settersutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	GcloudProject       = "gcloud.core.project"
	GcloudProjectNumber = "gcloud.project.projectNumber"
)

type AutoSet struct {
	// Writer is the output writer
	Writer io.Writer

	// PackagePath is the path of the package to apply auto-setters
	PackagePath string
}

// PerformAutoSetters auto-fills the setter values from the local environment
// in the target path for the setters which are not already set previously
// Auto setters are applied in the following order of precedence
// 1. Setter values from the parent package
// 2. Setter values from the environment variables
// 3. Setter values from gcloud configs
// The auto setters are applied for all the subpackages with in the directory
// tree of input PackagePath
// Only the setters which are NOT set locally (identified by isSet flag in setter
// definition of Kptfile), are only set by this operation
func (a AutoSet) PerformAutoSetters() error {
	// auto-fill setter values from parent package
	if err := a.SetInheritedSetters(); err != nil {
		return err
	}

	// auto-fill setters from environment
	if err := a.SetEnvAutoSetters(); err != nil {
		return err
	}

	// auto-fill setters from gcloud config
	if err := a.SetGcloudAutoSetters(); err != nil {
		return err
	}

	return nil
}

// SetGcloudAutoSetters auto-fills setters from gcloud config
func (a AutoSet) SetGcloudAutoSetters() error {
	gcloudConfig := []string{"compute.region", "compute.zone", "core.project"}
	for _, c := range gcloudConfig {
		gcloudCmd := exec.Command("gcloud",
			"config", "list", "--format", fmt.Sprintf("value(%s)", c))
		b, err := gcloudCmd.Output()
		if err != nil {
			// don't fail if gcloud fails -- it may not be installed or have this config property
			continue
		}
		v := strings.TrimSpace(string(b))
		if v == "" {
			// don't replace values that aren't set - stick with the defaults as defined in the manifest
			continue
		}

		err = SetV2AutoSetter(fmt.Sprintf("gcloud.%s", c), v, a.PackagePath, a.Writer)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetV2AutoSetter sets the input auto setter recursively in all the sub-packages of root
// Sets GcloudProjectNumber as well, if input setter is GcloudProject
func SetV2AutoSetter(name, value, root string, w io.Writer) error {
	resourcePackagesPaths, err := pathutil.DirsWithFile(root, kptfile.KptFileName, true)
	if err != nil {
		return err
	}
	for _, resourcesPath := range resourcePackagesPaths {
		if !DefExists(resourcesPath, name) || isSet(name, filepath.Join(resourcesPath, kptfile.KptFileName)) {
			continue
		}
		fs := &settersutil.FieldSetter{
			Name:            name,
			Value:           value,
			SetBy:           "kpt",
			OpenAPIPath:     filepath.Join(resourcesPath, kptfile.KptFileName),
			OpenAPIFileName: kptfile.KptFileName,
			ResourcesPath:   resourcesPath,
			IsSet:           true,
		}
		format := "automatically set %d field(s) for setter %q to value %q in package %q derived from gcloud config\n"
		count, err := fs.Set()
		if err != nil {
			fmt.Fprintf(w, "failed to set %q automatically in package %q with error: %s\n", name, resourcesPath, err.Error())
		} else {
			fmt.Fprintf(w, format, count, name, value, resourcesPath)
		}
		if name == GcloudProject && count > 0 {
			projectID := value
			// set the projectNumber if the projectID is set
			projectNumber, err := GetProjectNumberFromProjectID(projectID)
			if err != nil {
				return err
			}
			if DefExists(resourcesPath, GcloudProjectNumber) && projectNumber != "" {
				if isSet(GcloudProjectNumber, filepath.Join(resourcesPath, kptfile.KptFileName)) {
					continue
				}
				fs = &settersutil.FieldSetter{
					Name:            GcloudProjectNumber,
					Value:           projectNumber,
					SetBy:           "kpt",
					OpenAPIPath:     filepath.Join(resourcesPath, kptfile.KptFileName),
					OpenAPIFileName: kptfile.KptFileName,
					ResourcesPath:   resourcesPath,
					IsSet:           true,
				}
				count, err := fs.Set()
				if err != nil {
					fmt.Fprintf(w, "failed setting auto-setter %q in package %q with error: %s\n", GcloudProjectNumber, resourcesPath, err.Error())
				} else {
					fmt.Fprintf(w, format, count, GcloudProjectNumber, projectNumber, resourcesPath)
				}
			}
		}
	}
	return nil
}

var GetProjectNumberFromProjectID = func(projectID string) (string, error) {
	gcloudCmd := exec.Command("gcloud",
		"projects", "describe", projectID, "--format", "value(projectNumber)")
	b, err := gcloudCmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get project number for %s, please verify gcloud "+
			"credentials are valid and try again", projectID)
	}
	return strings.TrimSpace(string(b)), nil
}

// SetEnvAutoSetters auto-fills setters from the environment
func (a AutoSet) SetEnvAutoSetters() error {
	resourcePackagesPaths, err := pathutil.DirsWithFile(a.PackagePath, kptfile.KptFileName, true)
	if err != nil {
		return err
	}
	envVariables := environmentVariables()
	for i := range envVariables {
		e := envVariables[i]
		if !strings.HasPrefix(e, "KPT_SET_") {
			continue
		}
		parts := strings.SplitN(e, "=", 2)
		if len(parts) < 2 {
			continue
		}
		k, v := strings.TrimPrefix(parts[0], "KPT_SET_"), parts[1]

		for _, resourcesPath := range resourcePackagesPaths {
			if !DefExists(resourcesPath, k) || isSet(k, filepath.Join(resourcesPath, kptfile.KptFileName)) {
				continue
			}

			fs := &settersutil.FieldSetter{
				Name:            k,
				Value:           v,
				SetBy:           "kpt",
				ResourcesPath:   resourcesPath,
				OpenAPIPath:     filepath.Join(resourcesPath, kptfile.KptFileName),
				OpenAPIFileName: kptfile.KptFileName,
				IsSet:           true,
			}
			count, err := fs.Set()
			if err != nil {
				fmt.Fprintf(a.Writer, "failed to set %q automatically in package %q with error: %s\n", k, resourcesPath, err.Error())
			} else {
				format := "automatically set %d field(s) for setter %q to value %q in package %q derived from environment\n"
				fmt.Fprintf(a.Writer, format, count, k, v, resourcesPath)
			}
		}
	}
	return nil
}

var environmentVariables = os.Environ

// SetInheritedSetters traverses the absolute parentPath all the way to the root
// of file system to find a kpt package with Kptfile and auto-fills the setter
// values in targetPath and all its subpackages
func (a AutoSet) SetInheritedSetters() error {
	// get all the subpackage paths(including itself) in PackagePath
	targetPackagesPaths, err := pathutil.DirsWithFile(a.PackagePath, kptfile.KptFileName, true)
	if err != nil {
		return err
	}

	// for each subpackage find its parent on local and inherit the setter values
	for _, targetPath := range targetPackagesPaths {
		parentKptfilePath, err := parentDirWithKptfile(filepath.Dir(targetPath))
		if err != nil {
			return err
		}

		if parentKptfilePath == "" {
			// continue to next package if there is no parent package for current targetPath
			continue
		}

		targetPkgRefs, err := openapi.DefinitionRefs(filepath.Join(targetPath, kptfile.KptFileName))
		if err != nil {
			return err
		}

		// for each setter in target path, derive the setter values from parent package
		// openAPI schema definitions and set them
		for _, ref := range targetPkgRefs {
			err := a.setInheritedSettersForPkg(targetPath, parentKptfilePath, ref)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// setInheritedSettersForPkg inherits the setter value of setterRef to pkgPath from parentKptfilePath
func (a AutoSet) setInheritedSettersForPkg(pkgPath, parentKptfilePath, setterRef string) error {
	sc, err := openapi.SchemaFromFile(parentKptfilePath)
	if err != nil {
		return err
	}
	sch := sc.Definitions[setterRef]
	cliExt, err := setters2.GetExtFromSchema(&sch)
	if cliExt == nil || cliExt.Setter == nil || err != nil {
		// if the ref doesn't exist in global schema or if it is not a setter
		// continue, as there might be setters which are not present global schema
		return nil
	}
	kptfilePath := filepath.Join(pkgPath, kptfile.KptFileName)
	if isSet(cliExt.Setter.Name, kptfilePath) {
		// skip if the setter is already set on local
		return nil
	}

	fs := &settersutil.FieldSetter{
		Name:            cliExt.Setter.Name,
		Value:           cliExt.Setter.Value,
		ListValues:      cliExt.Setter.ListValues,
		OpenAPIPath:     kptfilePath,
		OpenAPIFileName: kptfile.KptFileName,
		ResourcesPath:   pkgPath,
		// turn isSet to true on child Kptfile iff the value derived from parent
		// is set by user, don't set isSet to true for inheriting default values from parent
		IsSet: isSet(cliExt.Setter.Name, parentKptfilePath),
	}

	count, err := fs.Set()
	if err != nil {
		fmt.Fprintf(a.Writer, "failed to set %q automatically in package %q with error: %s\n", cliExt.Setter.Name, pkgPath, err.Error())
	} else {

		format := "automatically set %d field(s) for setter %q to value %q in package %q derived from parent %q\n"

		if len(cliExt.Setter.ListValues) == 0 {
			fmt.Fprintf(a.Writer, format, count, cliExt.Setter.Name, cliExt.Setter.Value, pkgPath, parentKptfilePath)
		} else {
			fmt.Fprintf(a.Writer, format, count, cliExt.Setter.Name, cliExt.Setter.ListValues, pkgPath, parentKptfilePath)
		}
	}
	return nil
}

// isSet checks the openAPI file and returns true iff the openAPI definition
// for for given setter has isSet flag set to true
func isSet(setterName, openAPIFile string) bool {
	b, err := ioutil.ReadFile(openAPIFile)
	if err != nil {
		return false
	}
	node, err := yaml.Parse(string(b))
	if err != nil {
		return false
	}
	isSetNode, err := node.Pipe(yaml.Lookup(
		openapi.SupplementaryOpenAPIFieldName,
		openapi.Definitions,
		fieldmeta.SetterDefinitionPrefix+setterName, setters2.K8sCliExtensionKey,
		"setter", "isSet"))
	if err != nil {
		return false
	}
	isSetVal, err := isSetNode.String()
	if err != nil {
		return false
	}
	return strings.TrimSpace(isSetVal) == "true"
}

// parentDirWithKptfile traverses the parentPath till the root of the file system
// and returns the Kptfile path first encountered
func parentDirWithKptfile(parentPath string) (string, error) {
	parentPath = filepath.Clean(parentPath)
	absParentPath, err := filepath.Abs(parentPath)
	if err != nil {
		return "", err
	}
	for {
		openAPIPath := filepath.Join(absParentPath, kptfile.KptFileName)
		_, err := os.Stat(openAPIPath)
		if !os.IsNotExist(err) {
			return openAPIPath, nil
		}
		// terminal condition: stop if there is no parent dir for absParentPath
		if absParentPath == filepath.Dir(absParentPath) {
			return "", nil
		}
		absParentPath = filepath.Dir(absParentPath)
	}
}

// DefExists returns true if the setterName exists in Kptfile definitions
func DefExists(resourcePath, setterName string) bool {
	sc, err := openapi.SchemaFromFile(filepath.Join(resourcePath, kptfile.KptFileName))
	if err != nil {
		return false
	}
	ref, err := spec.NewRef(fieldmeta.DefinitionsPrefix + fieldmeta.SetterDefinitionPrefix + setterName)
	if err != nil {
		return false
	}
	setter, _ := openapi.Resolve(&ref, sc)
	return setter != nil
}

// CheckForRequiredSetters takes the package path, checks if there is a KrmFile
// and checks if all the required setters are set
func CheckForRequiredSetters(path string) error {
	kptFilePath := filepath.Join(path, kptfile.KptFileName)
	_, err := os.Stat(kptFilePath)
	if err != nil {
		// if file is not readable or doesn't exist, exit without error
		// as there might be packages without KrmFile
		return nil
	}
	settersSchema, err := openapi.SchemaFromFile(kptFilePath)
	if err != nil {
		return err
	}
	if settersSchema == nil {
		// this happens when there is Kptfile but no setter definitions
		return nil
	}
	return setters2.CheckRequiredSettersSet(settersSchema)
}
