// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package taskrunner

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/restmapper"
)

// ExtractDeferredDiscoveryRESTMapper unwraps the provided RESTMapper
// interface to get access to the underlying DeferredDiscoveryRESTMapper
// that can be reset.
func ExtractDeferredDiscoveryRESTMapper(mapper meta.RESTMapper) (*restmapper.DeferredDiscoveryRESTMapper,
	error) {
	val := reflect.ValueOf(mapper)
	if val.Type().Kind() != reflect.Struct {
		return nil, fmt.Errorf("unexpected RESTMapper type: %s", val.Type().String())
	}
	fv := val.FieldByName("RESTMapper")
	ddRESTMapper, ok := fv.Interface().(*restmapper.DeferredDiscoveryRESTMapper)
	if !ok {
		return nil, fmt.Errorf("unexpected RESTMapper type")
	}
	return ddRESTMapper, nil
}
