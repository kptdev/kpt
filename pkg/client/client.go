// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// Client is the client to update object in the API server.
type Client struct {
	client     dynamic.Interface
	restMapper meta.RESTMapper
}

// NewClient returns a client to get and update an object.
func NewClient(d dynamic.Interface, mapper meta.RESTMapper) *Client {
	return &Client{
		client:     d,
		restMapper: mapper,
	}
}

// Update updates an object using dynamic client
func (uc *Client) Update(ctx context.Context, meta object.ObjMetadata, obj *unstructured.Unstructured, options *metav1.UpdateOptions) error {
	r, err := uc.resourceInterface(meta)
	if err != nil {
		return err
	}
	if options == nil {
		options = &metav1.UpdateOptions{}
	}
	_, err = r.Update(ctx, obj, *options)
	return err
}

// Get fetches the requested object into the input obj using dynamic client
func (uc *Client) Get(ctx context.Context, meta object.ObjMetadata) (*unstructured.Unstructured, error) {
	r, err := uc.resourceInterface(meta)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, meta.Name, metav1.GetOptions{})
}

func (uc *Client) resourceInterface(meta object.ObjMetadata) (dynamic.ResourceInterface, error) {
	mapping, err := uc.restMapper.RESTMapping(meta.GroupKind)
	if err != nil {
		return nil, err
	}
	namespacedClient := uc.client.Resource(mapping.Resource).Namespace(meta.Namespace)
	return namespacedClient, nil
}

// ReplaceOwningInventoryID updates the object owning inventory annotation
// to the new ID when the owning inventory annotation is either empty or the old ID.
// It returns true if the annotation is updated.
func ReplaceOwningInventoryID(obj *unstructured.Unstructured, oldID, newID string) (bool, error) {
	key := "config.k8s.io/owning-inventory"
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	val, found := annotations[key]
	if !found || val == oldID {
		annotations[key] = newID
		return true, updateAnnotations(obj, annotations)
	}
	return false, nil
}

// updateAnnotations updates .metadata.annotations field of obj to use the passed in annotations
// as well as updates the last-applied-configuration annotations.
func updateAnnotations(obj *unstructured.Unstructured, annotations map[string]string) error {
	u := getOriginalObj(obj)
	if u != nil {
		u.SetAnnotations(annotations)
		// Since the annotation is updated, we also need to update the
		// last applied configuration annotation.
		err := util.CreateOrUpdateAnnotation(true, u, scheme.DefaultJSONEncoder())
		obj.SetAnnotations(u.GetAnnotations())
		return err
	}
	obj.SetAnnotations(annotations)
	return nil
}

// UpdateLabelsAndAnnotations updates .metadata.labels and .metadata.annotations fields of obj to use
// the passed in labels and annotations.
// It also updates the last-applied-configuration annotations.
func UpdateLabelsAndAnnotations(obj *unstructured.Unstructured, labels, annotations map[string]string) error {
	u := getOriginalObj(obj)
	if u != nil {
		u.SetAnnotations(annotations)
		u.SetLabels(labels)
		// Since the annotation is updated, we also need to update the
		// last applied configuration annotation.
		err := util.CreateOrUpdateAnnotation(true, u, scheme.DefaultJSONEncoder())
		obj.SetLabels(u.GetLabels())
		obj.SetAnnotations(u.GetAnnotations())
		return err
	}
	obj.SetLabels(labels)
	obj.SetAnnotations(annotations)
	return nil
}

func getOriginalObj(obj *unstructured.Unstructured) *unstructured.Unstructured {
	annotations := obj.GetAnnotations()
	lastApplied, found := annotations[v1.LastAppliedConfigAnnotation]
	if !found {
		return nil
	}
	u := &unstructured.Unstructured{}
	err := json.Unmarshal([]byte(lastApplied), u)
	if err != nil {
		return nil
	}
	return u
}
