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
	"fmt"
	"sort"
	"strings"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	//configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	//"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConfigInjectionAnnotation = "kpt.dev/config-injection"
)

func injectionConditionType(object *fn.KubeObject) string {
	return strings.Join([]string{"config", "injection", object.GetKind(), object.GetName()}, ".")
}

type injectionPoint struct {
	file          string
	object        *fn.KubeObject
	conditionType string
	required      bool
	errors        []string
	injected      bool
	injectedName  string
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

func ensureConfigInjection(client client.Client,
	pv *api.PackageVariant,
	prr *porchapi.PackageRevisionResources) error {

	if prr.Spec.Resources == nil {
		return fmt.Errorf("nil resources found for PackageRevisionResources '%s/%s'", prr.Namespace, prr.Name)
	}

	injectionPoints, err := findInjectionPoints(prr)
	if err != nil {
		return err
	}

	err = validateInjectionPoints(injectionPoints)
	if err != nil {
		return err
	}

	err = injectResources(client, pv.Spec.Injectors, injectionPoints)
	if err != nil {
		return err
	}

	kptfile, err := getFileKubeObject(prr, "Kptfile", "Kptfile", "")
	if err != nil {
		return err
	}

	setInjectionPointConditionsAndGates(kptfile, injectionPoints)

	if len(pv.Spec.Injectors) == 0 {
		return nil
	}

	return nil
}

func findInjectionPoints(prr *porchapi.PackageRevisionResources) ([]*injectionPoint, error) {
	var injectionPoints []*injectionPoint
	for file, r := range prr.Spec.Resources {
		// Convert to KubeObjects for easier processing
		kos, err := fn.ParseKubeObjects([]byte(r))
		if err != nil {
			return nil, fmt.Errorf("%s: %s", file, err.Error())
		}

		// Loop through the resources and find all injection points
		for _, ko := range kos {
			ip := newInjectionPoint(file, ko)
			if ip != nil {
				injectionPoints = append(injectionPoints, ip)
			}
		}
	}
	return injectionPoints, nil
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

func injectResources(client client.Client, injectors []api.InjectionSelector, injectionPoints []*injectionPoint) error {
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
	sort.Slice(gates, func(i, j int) bool { return gates[i].ConditionType < gates[j].ConditionType })

	if gates != nil {
		info.ReadinessGates = gates
		err = kptfileKubeObject.SetNestedField(info, "info")
		if err != nil {
			return err
		}
	}

	// update the status conditions
	if conditions != nil {
		sort.Slice(conditions, func(i, j int) bool { return conditions[i].Type < conditions[j].Type })
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
