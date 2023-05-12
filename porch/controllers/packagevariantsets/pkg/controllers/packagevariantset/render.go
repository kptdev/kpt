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
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/google/cel-go/cel"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha2"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	// TODO: including this requires many dependency updates, at some point
	// we should do that so the CEL evaluation here is consistent with
	// K8s. There are a few other lines to uncomment in that case.
	//"k8s.io/apiserver/pkg/cel/library"
)

const (
	RepoDefaultVarName    = "repoDefault"
	PackageDefaultVarName = "packageDefault"
	UpstreamVarName       = "upstream"
	RepositoryVarName     = "repository"
	TargetVarName         = "target"
)

func renderPackageVariantSpec(ctx context.Context, pvs *api.PackageVariantSet, repoList *configapi.RepositoryList,
	upstreamPR *porchapi.PackageRevision, downstream pvContext) (*pkgvarapi.PackageVariantSpec, error) {

	spec := &pkgvarapi.PackageVariantSpec{
		Upstream: pvs.Spec.Upstream,
		Downstream: &pkgvarapi.Downstream{
			Repo:    downstream.repoDefault,
			Package: downstream.packageDefault,
		},
	}

	pvt := downstream.template
	if pvt == nil {
		return spec, nil
	}

	inputs, err := buildBaseInputs(upstreamPR, downstream)
	if err != nil {
		return nil, err
	}

	repo := downstream.repoDefault

	if pvt.Downstream != nil {
		if pvt.Downstream.Repo != nil && *pvt.Downstream.Repo != "" {
			repo = *pvt.Downstream.Repo
		}

		if pvt.Downstream.RepoExpr != nil && *pvt.Downstream.RepoExpr != "" {
			repo, err = evalExpr(*pvt.Downstream.RepoExpr, inputs)
			if err != nil {
				return nil, fmt.Errorf("template.downstream.repoExpr: %s", err.Error())
			}
		}

		spec.Downstream.Repo = repo
	}

	for _, r := range repoList.Items {
		if r.Name == repo {
			repoInput, err := objectToInput(&r)
			if err != nil {
				return nil, err
			}
			inputs[RepositoryVarName] = repoInput
			break
		}
	}

	if _, ok := inputs[RepositoryVarName]; !ok {
		return nil, fmt.Errorf("repository %q could not be loaded", repo)
	}

	if pvt.Downstream != nil {
		if pvt.Downstream.Package != nil && *pvt.Downstream.Package != "" {
			spec.Downstream.Package = *pvt.Downstream.Package
		}

		if pvt.Downstream.PackageExpr != nil && *pvt.Downstream.PackageExpr != "" {
			spec.Downstream.Package, err = evalExpr(*pvt.Downstream.PackageExpr, inputs)
			if err != nil {
				return nil, fmt.Errorf("template.downstream.packageExpr: %s", err.Error())
			}
		}
	}

	if pvt.AdoptionPolicy != nil {
		spec.AdoptionPolicy = *pvt.AdoptionPolicy
	}

	if pvt.DeletionPolicy != nil {
		spec.DeletionPolicy = *pvt.DeletionPolicy
	}
	spec.Labels, err = copyAndOverlayMapExpr("template.labelExprs", pvt.Labels, pvt.LabelExprs, inputs)
	if err != nil {
		return nil, err
	}

	spec.Annotations, err = copyAndOverlayMapExpr("template.annotationExprs", pvt.Annotations, pvt.AnnotationExprs, inputs)
	if err != nil {
		return nil, err
	}

	if pvt.PackageContext != nil {
		data, err := copyAndOverlayMapExpr("template.packageContext.dataExprs", pvt.PackageContext.Data, pvt.PackageContext.DataExprs, inputs)
		if err != nil {
			return nil, err
		}

		removeKeys, err := copyAndOverlayStringSlice("template.packageContext.removeKeyExprs", pvt.PackageContext.RemoveKeys,
			pvt.PackageContext.RemoveKeyExprs, inputs)
		if err != nil {
			return nil, err
		}
		spec.PackageContext = &pkgvarapi.PackageContext{
			Data:       data,
			RemoveKeys: removeKeys,
		}
	}

	for i, injTemplate := range pvt.Injectors {
		injector := pkgvarapi.InjectionSelector{
			Group:   injTemplate.Group,
			Version: injTemplate.Version,
			Kind:    injTemplate.Kind,
		}
		if injTemplate.Name != nil && *injTemplate.Name != "" {
			injector.Name = *injTemplate.Name
		}

		if injTemplate.NameExpr != nil && *injTemplate.NameExpr != "" {
			injector.Name, err = evalExpr(*injTemplate.NameExpr, inputs)
			if err != nil {
				return nil, fmt.Errorf("template.injectors[%d].nameExpr: %s", i, err.Error())
			}
		}

		spec.Injectors = append(spec.Injectors, injector)
	}

	if pvt.Pipeline != nil {
		pipeline := kptfilev1.Pipeline{}
		pipeline.Validators, err = renderFunctionTemplateList("template.pipeline.validators", pvt.Pipeline.Validators, inputs)
		if err != nil {
			return nil, err
		}
		pipeline.Mutators, err = renderFunctionTemplateList("template.pipeline.mutators", pvt.Pipeline.Mutators, inputs)
		if err != nil {
			return nil, err
		}
		if len(pipeline.Validators) > 0 || len(pipeline.Mutators) > 0 {
			spec.Pipeline = &pipeline
		}
	}

	return spec, nil
}

func renderFunctionTemplateList(field string, templateList []api.FunctionTemplate, inputs map[string]interface{}) ([]kptfilev1.Function, error) {
	var results []kptfilev1.Function
	for i, ft := range templateList {
		var err error
		f := ft.Function
		f.ConfigMap, err = copyAndOverlayMapExpr(fmt.Sprintf("%s[%d].configMapExprs", field, i), ft.ConfigMap, ft.ConfigMapExprs, inputs)
		if err != nil {
			return nil, err
		}

		results = append(results, f)
	}
	return results, nil
}

func buildBaseInputs(upstreamPR *porchapi.PackageRevision, downstream pvContext) (map[string]interface{}, error) {
	inputs := make(map[string]interface{}, 5)
	inputs[RepoDefaultVarName] = downstream.repoDefault
	inputs[PackageDefaultVarName] = downstream.packageDefault

	upstreamInput, err := objectToInput(upstreamPR)
	if err != nil {
		return nil, err
	}
	inputs[UpstreamVarName] = upstreamInput

	if downstream.object != nil {
		targetInput, err := objectToInput(downstream.object)
		if err != nil {
			return nil, err
		}
		inputs[TargetVarName] = targetInput
	} else {
		inputs[TargetVarName] = map[string]string{
			"repo":    downstream.repoDefault,
			"package": downstream.packageDefault,
		}
	}

	return inputs, nil
}

func objectToInput(obj interface{}) (map[string]interface{}, error) {

	result := make(map[string]interface{})

	uo, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	u := unstructured.Unstructured{Object: uo}

	//TODO: allow an administrator-configurable allow list of fields,
	// on a per-GVK basis
	result["name"] = u.GetName()
	result["namespace"] = u.GetNamespace()
	result["labels"] = u.GetLabels()
	result["annotations"] = u.GetAnnotations()

	return result, nil
}

func copyAndOverlayMapExpr(fieldName string, inMap map[string]string, mapExprs []api.MapExpr, inputs map[string]interface{}) (map[string]string, error) {
	outMap := make(map[string]string, len(inMap))
	for k, v := range inMap {
		outMap[k] = v
	}

	var err error
	for i, me := range mapExprs {
		var k, v string
		if me.Key != nil {
			k = *me.Key
		}
		if me.KeyExpr != nil {
			k, err = evalExpr(*me.KeyExpr, inputs)
			if err != nil {
				return nil, fmt.Errorf("%s[%d].keyExpr: %s", fieldName, i, err.Error())
			}
		}
		if me.Value != nil {
			v = *me.Value
		}
		if me.ValueExpr != nil {
			v, err = evalExpr(*me.ValueExpr, inputs)
			if err != nil {
				return nil, fmt.Errorf("%s[%d].valueExpr: %s", fieldName, i, err.Error())
			}
		}
		outMap[k] = v
	}

	if len(outMap) == 0 {
		return nil, nil
	}

	return outMap, nil
}

func copyAndOverlayStringSlice(fieldName string, in, exprs []string, inputs map[string]interface{}) ([]string, error) {
	outMap := make(map[string]bool, len(in)+len(exprs))

	for _, v := range in {
		outMap[v] = true
	}
	for i, e := range exprs {
		v, err := evalExpr(e, inputs)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %s", fieldName, i, err.Error())
		}
		outMap[v] = true
	}

	if len(outMap) == 0 {
		return nil, nil
	}

	var out []string
	for k := range outMap {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

func evalExpr(expr string, inputs map[string]interface{}) (string, error) {
	prog, err := compileExpr(expr)
	if err != nil {
		return "", err
	}

	val, _, err := prog.Eval(inputs)
	if err != nil {
		return "", err
	}

	result, err := val.ConvertToNative(reflect.TypeOf(""))
	if err != nil {
		return "", err
	}

	s, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("expression returned non-string value: %v", result)
	}

	return s, nil
}

// compileExpr returns a compiled CEL expression.
func compileExpr(expr string) (cel.Program, error) {
	var opts []cel.EnvOption
	opts = append(opts, cel.HomogeneousAggregateLiterals())
	opts = append(opts, cel.EagerlyValidateDeclarations(true), cel.DefaultUTCTimeZone(true))
	// TODO: uncomment after updating to latest k8s
	//opts = append(opts, library.ExtensionLibs...)
	opts = append(opts, cel.Variable(RepoDefaultVarName, cel.StringType))
	opts = append(opts, cel.Variable(PackageDefaultVarName, cel.StringType))
	opts = append(opts, cel.Variable(UpstreamVarName, cel.DynType))
	opts = append(opts, cel.Variable(TargetVarName, cel.DynType))
	opts = append(opts, cel.Variable(RepositoryVarName, cel.DynType))

	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, err
	}

	ast, issues := env.Compile(expr)
	if issues != nil {
		return nil, issues.Err()
	}

	_, err = cel.AstToCheckedExpr(ast)
	if err != nil {
		return nil, err
	}
	return env.Program(ast,
		cel.EvalOptions(cel.OptOptimize),
		// TODO: uncomment after updating to latest k8s
		//cel.OptimizeRegex(library.ExtensionLibRegexOptimizations...),
	)
}
