// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package krmfn

import (
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const ModuleName = "krmfn.star"

var Module = &starlarkstruct.Module{
	Name: "krmfn",
	Members: starlark.StringDict{
		"match_gvk":       starlark.NewBuiltin("match_gvk", matchGVK),
		"match_name":      starlark.NewBuiltin("match_name", matchName),
		"match_namespace": starlark.NewBuiltin("match_namespace", matchNamespace),
	},
}

func matchGVK(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var resource starlark.Value
	var apiVersion, kind string
	if err := starlark.UnpackPositionalArgs("match_gvk", args, kwargs, 3,
		&resource, &apiVersion, &kind); err != nil {
		return nil, err
	}
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}
	rn, err := yaml.Parse(resource.String())
	if err != nil {
		return nil, err
	}
	meta, err := rn.GetMeta()
	if err != nil {
		return nil, err
	}
	parsedGV, err := schema.ParseGroupVersion(meta.APIVersion)
	if err != nil {
		return starlark.False, nil
	}
	return starlark.Bool(parsedGV.Group == gv.Group &&
		parsedGV.Version == gv.Version &&
		meta.Kind == kind), nil
}

func matchName(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var resource starlark.Value
	var name string
	if err := starlark.UnpackPositionalArgs("match_name", args, kwargs, 2,
		&resource, &name); err != nil {
		return nil, err
	}
	rn, err := yaml.Parse(resource.String())
	if err != nil {
		return nil, err
	}
	meta, err := rn.GetMeta()
	if err != nil {
		return nil, err
	}
	return starlark.Bool(meta.Name == name), nil
}

func matchNamespace(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var resource starlark.Value
	var namespace string
	if err := starlark.UnpackPositionalArgs("match_namespace", args, kwargs, 2,
		&resource, &namespace); err != nil {
		return nil, err
	}
	rn, err := yaml.Parse(resource.String())
	if err != nil {
		return nil, err
	}
	meta, err := rn.GetMeta()
	if err != nil {
		return nil, err
	}
	return starlark.Bool(meta.Namespace == namespace), nil
}
