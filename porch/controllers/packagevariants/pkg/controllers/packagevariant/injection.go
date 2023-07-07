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

package packagevariant

import (
	"context"
	"fmt"
	"path/filepath"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConfigInjectionAnnotation  = "kpt.dev/config-injection"
	InjectedResourceAnnotation = "kpt.dev/injected-resource"
)

func injectionConditionType(object *fn.KubeObject) string {
	return strings.Join([]string{"config", "injection", object.GetKind(), object.GetName()}, ".")
}

type injectableField struct {
	Group string
	Kind  string
	Field string
}

// TODO: consider giving the admin control over this
var allowedInjectionFields = []injectableField{
	{Group: "", Kind: "ConfigMap", Field: "data"},
	{Group: "*", Kind: "*", Field: "spec"},
}

type injectionPoint struct {
	file               string
	object             *fn.KubeObject
	conditionType      string
	required           bool
	errors             []string
	injected           bool
	injectedName       string
	inClusterResources []*unstructured.Unstructured
}

func newInjectionPoint(file string, object *fn.KubeObject) *injectionPoint {
	annotations := object.GetAnnotations()
	annotation, ok := annotations[ConfigInjectionAnnotation]
	if !ok {
		return nil
	}
	ip := &injectionPoint{
		file:          file,
		object:        object,
		conditionType: injectionConditionType(object),
	}
	if annotation == "required" {
		ip.required = true
	} else if annotation == "optional" {
		ip.required = false
	} else {
		ip.errors = append(ip.errors, fmt.Sprintf("%s: %s/%s has invalid %q annotation value of %q",
			file, object.GetKind(), object.GetName(), ConfigInjectionAnnotation, annotation))
	}
	return ip
}

func ensureConfigInjection(ctx context.Context,
	c client.Client,
	pv *api.PackageVariant,
	prr *porchapi.PackageRevisionResources) error {

	files, err := parseFiles(prr)
	if err != nil {
		return err
	}

	injectionPoints := findInjectionPoints(files)

	err = validateInjectionPoints(injectionPoints)
	if err != nil {
		return err
	}

	injectResources(ctx, c, pv.Namespace, pv.Spec.Injectors, injectionPoints)

	// find which files need to be updated
	// this might do some files more than once, but that's ok
	for _, ip := range injectionPoints {
		if ip.injected {
			prr.Spec.Resources[ip.file] = kubeobjectsToYaml(files[ip.file])
		}
	}

	kptfile, err := getFileKubeObject(prr, "Kptfile", "Kptfile", "")
	if err != nil {
		return err
	}

	setInjectionPointConditionsAndGates(kptfile, injectionPoints)

	prr.Spec.Resources["Kptfile"] = kptfile.String()

	return nil
}

func kubeobjectsToYaml(kos fn.KubeObjects) string {
	var yamls []string
	for _, ko := range kos {
		yamls = append(yamls, ko.String())
	}
	return strings.Join(yamls, "---\n")
}

func parseFiles(prr *porchapi.PackageRevisionResources) (map[string]fn.KubeObjects, error) {
	result := make(map[string]fn.KubeObjects)
	for file, r := range prr.Spec.Resources {
		if !includeFile(file) {
			continue
		}

		// Convert to KubeObjects for easier processing
		kos, err := fn.ParseKubeObjects([]byte(r))
		if err != nil {
			return nil, fmt.Errorf("%s: %s", file, err.Error())
		}
		result[file] = kos
	}
	return result, nil
}

func findInjectionPoints(files map[string]fn.KubeObjects) []*injectionPoint {
	var injectionPoints []*injectionPoint
	for file, kos := range files {
		// Loop through the resources and find all injection points
		for _, ko := range kos {
			ip := newInjectionPoint(file, ko)
			if ip != nil {
				injectionPoints = append(injectionPoints, ip)
			}
		}
	}
	return injectionPoints
}

func validateInjectionPoints(injectionPoints []*injectionPoint) error {
	var allErrs []string
	// check if there are any duplicated condition types; this will be an error
	checkCT := make(map[string]*injectionPoint)
	for _, ip := range injectionPoints {
		allErrs = append(allErrs, ip.errors...)
		if origIP, ok := checkCT[ip.conditionType]; ok {
			allErrs = append(allErrs,
				fmt.Sprintf("duplicate injection conditionType %q (%s and %s)", ip.conditionType,
					origIP.file, ip.file))
		}
		checkCT[ip.conditionType] = ip
	}

	if len(allErrs) > 0 {
		return fmt.Errorf("errors in injection points: %s", strings.Join(allErrs, ", "))
	}

	return nil
}

func injectResources(ctx context.Context, c client.Client, namespace string, injectors []api.InjectionSelector, injectionPoints []*injectionPoint) {
	for _, ip := range injectionPoints {
		if len(injectors) == 0 {
			ip.errors = append(ip.errors, "no injectors defined")
			continue
		}
		if err := ip.loadInClusterResources(ctx, c, namespace); err != nil {
			ip.errors = append(ip.errors, err.Error())
			continue
		}
		ip.inject(injectors)
	}
}

func (ip *injectionPoint) inject(injectors []api.InjectionSelector) {
	if len(ip.inClusterResources) == 0 {
		ip.errors = append(ip.errors, fmt.Sprintf("no in-cluster resources of type %s.%s", ip.object.GetAPIVersion(), ip.object.GetKind()))
		return
	}

	for _, injector := range injectors {
		u := ip.matchSelector(injector)
		if u == nil {
			continue
		}

		ip.injectResource(u)
		break
	}
}

func (ip *injectionPoint) injectResource(u *unstructured.Unstructured) {
	ip.injected = true
	ip.injectedName = u.GetName()

	g, _ := fn.ParseGroupVersion(u.GetAPIVersion())

	for _, allowed := range allowedInjectionFields {
		if allowed.Group != "*" && allowed.Group != g {
			continue
		}
		if allowed.Kind != "*" && allowed.Kind != u.GetKind() {
			continue
		}

		obj, ok := u.Object[allowed.Field]
		if !ok {
			ip.injected = false
			ip.errors = append(ip.errors, fmt.Sprintf("field %q not found in resource %q", allowed.Field, u.GetName()))
			return
		}

		err := ip.object.SetNestedField(obj, allowed.Field)
		if err != nil {
			ip.injected = false
			ip.errors = append(ip.errors, err.Error())
			return
		}
		err = ip.object.SetAnnotation(InjectedResourceAnnotation, u.GetName())
		if err != nil {
			ip.injected = false
			ip.errors = append(ip.errors, err.Error())
			return
		}
		break
	}
}

func (ip *injectionPoint) matchSelector(injector api.InjectionSelector) *unstructured.Unstructured {
	// Check if this selector matches this in-package object
	g, v := fn.ParseGroupVersion(ip.object.GetAPIVersion())
	if injector.Group != nil && *injector.Group != g {
		return nil
	}
	if injector.Version != nil && *injector.Version != v {
		return nil
	}
	if injector.Kind != nil && *injector.Kind != ip.object.GetKind() {
		return nil
	}

	// This injector applies to this in-package object
	// So, check the in-cluster objects for a match
	// We already know the GVK matches, we just need to check
	// the names

	for _, u := range ip.inClusterResources {
		if u.GetName() == injector.Name {
			return u
		}
	}

	return nil
}

func setInjectionPointConditionsAndGates(kptfileKubeObject *fn.KubeObject, injectionPoints []*injectionPoint) error {
	var kptfile kptfilev1.KptFile
	err := kptfileKubeObject.As(&kptfile)
	if err != nil {
		return err
	}

	info := kptfile.Info
	if info == nil {
		info = &kptfilev1.PackageInfo{}
	}
	// generate a unique list of gates (append is not idempotent)
	gateMap := make(map[string]bool)
	for _, gate := range info.ReadinessGates {
		gateMap[gate.ConditionType] = true
	}

	status := kptfile.Status
	if status == nil {
		status = &kptfilev1.Status{}
	}
	conditions := convertConditionsToMeta(status.Conditions)
	// set a condition for each injection point
	for _, ip := range injectionPoints {
		if ip.required {
			gateMap[ip.conditionType] = true
		}
		var condStatus metav1.ConditionStatus
		condStatus = "False"
		condReason := "NoResourceSelected"
		condMessage := "no resource matched any injection selector for this injection point"
		if len(ip.errors) > 0 {
			condMessage = strings.Join(ip.errors, ", ")
		}
		if ip.injected {
			condStatus = "True"
			condReason = "ConfigInjected"
			condMessage = fmt.Sprintf("injected resource %q from cluster", ip.injectedName)
		}

		meta.SetStatusCondition(&conditions, metav1.Condition{
			Type:    ip.conditionType,
			Status:  condStatus,
			Reason:  condReason,
			Message: condMessage,
		})

	}

	// update the readiness gates
	// TODO: this loses comments right now, fix that
	var gates []kptfilev1.ReadinessGate
	for k := range gateMap {
		gates = append(gates, kptfilev1.ReadinessGate{ConditionType: k})
	}
	sort.SliceStable(gates, func(i, j int) bool { return gates[i].ConditionType < gates[j].ConditionType })

	if gates != nil {
		info.ReadinessGates = gates
		err = kptfileKubeObject.SetNestedField(info, "info")
		if err != nil {
			return err
		}
	}

	// update the status conditions
	if conditions != nil {
		sort.SliceStable(conditions, func(i, j int) bool { return conditions[i].Type < conditions[j].Type })
		status.Conditions = convertConditionsFromMetaToKptfile(conditions)
		err = kptfileKubeObject.SetNestedField(status, "status")
		if err != nil {
			return err
		}
	}

	return nil
}

func convertConditionsFromMetaToKptfile(conditions []metav1.Condition) []kptfilev1.Condition {
	var result []kptfilev1.Condition
	for _, c := range conditions {
		result = append(result, kptfilev1.Condition{
			Type:    c.Type,
			Reason:  c.Reason,
			Status:  kptfilev1.ConditionStatus(c.Status),
			Message: c.Message,
		})
	}
	return result
}

func convertConditionsToMeta(conditions []kptfilev1.Condition) []metav1.Condition {
	var result []metav1.Condition
	for _, c := range conditions {
		result = append(result, metav1.Condition{
			Type:    c.Type,
			Reason:  c.Reason,
			Status:  metav1.ConditionStatus(c.Status),
			Message: c.Message,
		})
	}
	return result
}

var matchResourceContents = append(kio.MatchAll, kptfilev1.KptFileName)

// TODO: Move to a utility function
// includeFile checks if the file should be parsed for resources
func includeFile(path string) bool {
	for _, m := range matchResourceContents {
		// Only use the filename for the check for whether we should
		// include the file.
		f := filepath.Base(path)
		if matched, err := filepath.Match(m, f); err == nil && matched {
			return true
		}
	}
	return false
}

func (ip *injectionPoint) loadInClusterResources(ctx context.Context, c client.Client, namespace string) error {
	uList := &unstructured.UnstructuredList{}
	group, version := fn.ParseGroupVersion(ip.object.GetAPIVersion())
	uList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    ip.object.GetKind(),
	})

	opts := []client.ListOption{client.InNamespace(namespace)}
	if err := c.List(ctx, uList, opts...); err != nil {
		return err
	}

	for _, u := range uList.Items {
		ip.inClusterResources = append(ip.inClusterResources, u.DeepCopy())
	}

	return nil
}
