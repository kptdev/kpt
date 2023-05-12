// Copyright 2023 The kpt Authors
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

package packagevariantset

import (
	"fmt"
	"strings"

	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha2"
)

func validatePackageVariantSet(pvs *api.PackageVariantSet) []error {
	var allErrs []error
	if pvs.Spec.Upstream == nil {
		allErrs = append(allErrs, fmt.Errorf("spec.upstream is a required field"))
	} else {
		if pvs.Spec.Upstream.Package == "" {
			allErrs = append(allErrs, fmt.Errorf("spec.upstream.package is a required field"))
		}
		if pvs.Spec.Upstream.Repo == "" {
			allErrs = append(allErrs, fmt.Errorf("spec.upstream.repo is a required field"))
		}
		if pvs.Spec.Upstream.Revision == "" {
			allErrs = append(allErrs, fmt.Errorf("spec.upstream.revision is a required field"))
		}
	}

	if len(pvs.Spec.Targets) == 0 {
		allErrs = append(allErrs, fmt.Errorf("must specify at least one item in spec.targets"))
	}
	for i, target := range pvs.Spec.Targets {
		allErrs = append(allErrs, validateTarget(i, target)...)
	}

	return allErrs
}

func validateTarget(i int, target api.Target) []error {
	var allErrs []error
	count := 0
	if target.Repositories != nil {
		count++

		if len(target.Repositories) == 0 {
			allErrs = append(allErrs, fmt.Errorf("spec.targets[%d].repositories must not be an empty list if specified", i))
		}

		for j, rt := range target.Repositories {
			if rt.Name == "" {
				allErrs = append(allErrs, fmt.Errorf("spec.targets[%d].repositories[%d].name cannot be empty", i, j))
			}

			for k, pn := range rt.PackageNames {
				if pn == "" {
					allErrs = append(allErrs, fmt.Errorf("spec.targets[%d].repositories[%d].packageNames[%d] cannot be empty", i, j, k))
				}
			}
		}
	}

	if target.RepositorySelector != nil {
		count++
	}

	if target.ObjectSelector != nil {
		count++
		if target.ObjectSelector.APIVersion == "" {
			allErrs = append(allErrs, fmt.Errorf("spec.targets[%d].objectselector.apiVersion cannot be empty", i))
		}
		if target.ObjectSelector.Kind == "" {
			allErrs = append(allErrs, fmt.Errorf("spec.targets[%d].objectselector.kind cannot be empty", i))
		}
	}

	if count != 1 {
		allErrs = append(allErrs, fmt.Errorf("spec.targets[%d] must specify one of `repositories`, `repositorySelector`, or `objectSelector`", i))
	}

	if target.Template == nil {
		return allErrs
	}

	return append(allErrs, validateTemplate(target.Template, fmt.Sprintf("spec.targets[%d].template", i))...)
}

func validateTemplate(template *api.PackageVariantTemplate, field string) []error {
	var allErrs []error
	if template.AdoptionPolicy != nil && *template.AdoptionPolicy != pkgvarapi.AdoptionPolicyAdoptNone &&
		*template.AdoptionPolicy != pkgvarapi.AdoptionPolicyAdoptExisting {
		allErrs = append(allErrs, fmt.Errorf("%s.adoptionPolicy can only be %q or %q", field,
			pkgvarapi.AdoptionPolicyAdoptNone, pkgvarapi.AdoptionPolicyAdoptExisting))
	}

	if template.DeletionPolicy != nil && *template.DeletionPolicy != pkgvarapi.DeletionPolicyOrphan &&
		*template.DeletionPolicy != pkgvarapi.DeletionPolicyDelete {
		allErrs = append(allErrs, fmt.Errorf("%s.deletionPolicy can only be %q or %q", field,
			pkgvarapi.DeletionPolicyOrphan, pkgvarapi.DeletionPolicyDelete))
	}

	if template.Downstream != nil {
		if template.Downstream.Repo != nil && template.Downstream.RepoExpr != nil {
			allErrs = append(allErrs, fmt.Errorf("%s may specify only one of `downstream.repo` and `downstream.repoExpr`", field))
		}
		if template.Downstream.Package != nil && template.Downstream.PackageExpr != nil {
			allErrs = append(allErrs, fmt.Errorf("%s may specify only one of `downstream.package` and `downstream.packageExpr`", field))
		}
	}

	if template.LabelExprs != nil {
		allErrs = append(allErrs, validateMapExpr(template.LabelExprs, fmt.Sprintf("%s.labelExprs", field))...)
	}

	if template.AnnotationExprs != nil {
		allErrs = append(allErrs, validateMapExpr(template.AnnotationExprs, fmt.Sprintf("%s.annotationExprs", field))...)
	}

	if template.PackageContext != nil && template.PackageContext.DataExprs != nil {
		allErrs = append(allErrs, validateMapExpr(template.PackageContext.DataExprs, fmt.Sprintf("%s.packageContext.dataExprs", field))...)
	}

	for i, injector := range template.Injectors {
		if injector.Name != nil && injector.NameExpr != nil {
			allErrs = append(allErrs, fmt.Errorf("%s.injectors[%d] may specify only one of `name` and `nameExpr`", field, i))
		}

		if injector.Name == nil && injector.NameExpr == nil {
			allErrs = append(allErrs, fmt.Errorf("%s.injectors[%d] must specify either `name` or `nameExpr`", field, i))
		}
	}

	if template.Pipeline != nil {
		for i, f := range template.Pipeline.Validators {
			allErrs = append(allErrs, validateFunction(&f, fmt.Sprintf("%s.pipeline.validators[%d]", field, i))...)
		}
		for i, f := range template.Pipeline.Mutators {
			allErrs = append(allErrs, validateFunction(&f, fmt.Sprintf("%s.pipeline.mutators[%d]", field, i))...)
		}
	}

	return allErrs
}

func validateMapExpr(m []api.MapExpr, fieldName string) []error {
	var allErrs []error
	for j, me := range m {
		if me.Key != nil && me.KeyExpr != nil {
			allErrs = append(allErrs, fmt.Errorf("%s[%d] may specify only one of `key` and `keyExpr`", fieldName, j))
		}
		if me.Value != nil && me.ValueExpr != nil {
			allErrs = append(allErrs, fmt.Errorf("%s[%d] may specify only one of `value` and `valueExpr`", fieldName, j))
		}
	}

	return allErrs
}

func validateFunction(f *api.FunctionTemplate, field string) []error {
	var allErrs []error
	if f.Image == "" {
		allErrs = append(allErrs, fmt.Errorf("%s.image must not be empty", field))
	}
	if strings.Contains(f.Name, ".") {
		allErrs = append(allErrs, fmt.Errorf("%s.name must not contain '.'", field))
	}
	return allErrs
}

func combineErrors(errs []error) string {
	var errMsgs []string
	for _, e := range errs {
		if e.Error() != "" {
			errMsgs = append(errMsgs, e.Error())
		}
	}
	return strings.Join(errMsgs, "; ")
}
