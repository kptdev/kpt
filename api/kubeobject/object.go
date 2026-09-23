// Copyright 2022,2025-2026 The kpt Authors
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

package kubeobject

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/kptdev/kpt/api/kubeobject/node"
	schema "github.com/kptdev/kpt/api/schema/v1"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// KubeObject presents a k8s object.
type KubeObject struct {
	SubObject
}

// LineComment returns the line comment, if the target field exist and a
// potential error.
func (o *KubeObject) LineComment(fields ...string) (string, bool, error) {
	rn, found, err := o.obj.GetRNode(fields...)
	if !found || err != nil {
		return "", found, err
	}
	return rn.YNode().LineComment, true, nil
}

// HeadComment returns the head comment, if the target field exist and a
// potential error.
func (o *KubeObject) HeadComment(fields ...string) (string, bool, error) {
	rn, found, err := o.obj.GetRNode(fields...)
	if !found || err != nil {
		return "", found, err
	}
	return rn.YNode().HeadComment, true, nil
}

func (o *KubeObject) SetLineComment(comment string, fields ...string) error {
	rn, found, err := o.obj.GetRNode(fields...)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("can't set line comment because the field doesn't exist")
	}
	rn.YNode().LineComment = comment
	return nil
}

func (o *KubeObject) SetHeadComment(comment string, fields ...string) error {
	rn, found, err := o.obj.GetRNode(fields...)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("can't set head comment because the field doesn't exist")
	}
	rn.YNode().HeadComment = comment
	return nil
}

// NewFromTypedObject construct a KubeObject from a typed object (e.g. corev1.Pod)
func NewFromTypedObject(v any) (*KubeObject, error) {
	kind := reflect.ValueOf(v).Kind()
	if kind == reflect.Pointer {
		kind = reflect.TypeOf(v).Elem().Kind()
	}
	var err error
	var m *node.MapVariant
	switch kind {
	case reflect.Struct, reflect.Map:
		m, err = node.TypedObjectToMapVariant(v)
	case reflect.Slice:
		return nil, fmt.Errorf(
			"the typed object should be of a reflect.Struct or reflect.Map, got reflect.Slice")
	}
	if err != nil {
		return nil, err
	}
	return asKubeObject(m), nil
}

// ShortString provides a human-readable information for the KubeObject Identifier in the form of GVKNN.
func (o *KubeObject) ShortString() string {
	return fmt.Sprintf("Resource(apiVersion=%v, kind=%v, namespace=%v, name=%v)",
		o.GetAPIVersion(), o.GetKind(), o.GetNamespace(), o.GetName())
}

// resourceIdentifier returns the resource identifier including apiVersion, kind,
// namespace and name.
func (o *KubeObject) resourceIdentifier() *yaml.ResourceIdentifier {
	apiVersion := o.GetAPIVersion()
	kind := o.GetKind()
	name := o.GetName()
	ns := o.GetNamespace()
	return &yaml.ResourceIdentifier{
		TypeMeta: yaml.TypeMeta{
			APIVersion: apiVersion,
			Kind:       kind,
		},
		NameMeta: yaml.NameMeta{
			Name:      name,
			Namespace: ns,
		},
	}
}

// PackageScopeUniqueID is a key that uniquely identifies a resource in a package,
// even if there are multiple resources with the same GVKNN.
type PackageScopeUniqueID struct {
	yaml.ResourceIdentifier
	Path  string
	Index int
}

func (id PackageScopeUniqueID) String() string {
	return fmt.Sprintf("%s|%s|%s|%s|%s:%d",
		id.Kind, id.Name, id.Namespace, id.APIVersion, id.Path, id.Index)
}

// GetPackageScopeUniqueID returns with a key that uniquely identifies a resource in a package,
// even if there are multiple resources with the same GVKNN.
func (o *KubeObject) GetPackageScopeUniqueID() PackageScopeUniqueID {
	return PackageScopeUniqueID{
		ResourceIdentifier: *o.resourceIdentifier(),
		Path:               o.PathAnnotation(),
		Index:              o.IndexAnnotation(),
	}
}

// GroupVersionKind returns the schema.GroupVersionKind for the specified object.
func (o *KubeObject) GroupVersionKind() schema.GroupVersionKind {
	gv, err := schema.ParseGroupVersion(o.GetAPIVersion())
	if err != nil {
		return schema.GroupVersionKind{}
	}
	gvk := gv.WithKind(o.GetKind())
	return gvk
}

// GroupKind returns the schema.GroupKind for the specified object.
func (o *KubeObject) GroupKind() schema.GroupKind {
	return o.GroupVersionKind().GroupKind()
}

// HasSameID returns true if the two KubeObjects has the same (Group, Version, Kind, Namespace, Name)
func (o *KubeObject) HasSameID(b *KubeObject) bool {
	return *o.resourceIdentifier() == *b.resourceIdentifier()
}

// IsGroupVersionKind compares the given group, version, and kind with KubeObject's apiVersion and Kind.
func (o *KubeObject) IsGroupVersionKind(gvk schema.GroupVersionKind) bool {
	return o.GroupVersionKind() == gvk
}

// IsGroupKind compares the given group and kind with KubeObject's apiVersion and Kind.
func (o *KubeObject) IsGroupKind(gk schema.GroupKind) bool {
	return o.GroupKind() == gk
}

// IsGVK compares the given group, version, and kind with KubeObject's apiVersion and Kind.
// It only matches on specified arguments, for example if the group is empty this will match any group.
//
// Deprecated: Prefer exact matching with IsGroupVersionKind or IsGroupKind
func (o *KubeObject) IsGVK(group, version, kind string) bool {
	gvk := o.GroupVersionKind()
	if gvk.Kind != "" && kind != "" && gvk.Kind != kind {
		return false
	}
	if gvk.Group != "" && group != "" && gvk.Group != group {
		return false
	}
	if gvk.Version != "" && version != "" && gvk.Version != version {
		return false
	}
	return true
}

// IsLocalConfig checks the "config.kubernetes.io/local-config" field to tell
// whether a KRM resource will be skipped by `kpt live apply` or not.
func (o *KubeObject) IsLocalConfig() bool {
	isLocalConfig := o.GetAnnotation(KptLocalConfig)
	if isLocalConfig == "" || isLocalConfig == "false" {
		return false
	}
	return true
}

// IsLocalConfig determines whether a KubeObject (or KRM resource) has the config.kubernetes.io/local-config: true annotation
func IsLocalConfig(o *KubeObject) bool {
	return o.IsLocalConfig()
}

func (o *KubeObject) GetAPIVersion() string {
	apiVersion, _, _ := o.obj.GetNestedString("apiVersion")
	return apiVersion
}

func (o *KubeObject) SetAPIVersion(apiVersion string) error {
	return o.obj.SetNestedString(apiVersion, "apiVersion")
}

func (o *KubeObject) GetKind() string {
	kind, _, _ := o.obj.GetNestedString("kind")
	return kind
}

func (o *KubeObject) SetKind(kind string) error {
	return o.SetNestedField(kind, "kind")
}

func (o *KubeObject) GetName() string {
	s, _, _ := o.obj.GetNestedString("metadata", "name")
	return s
}

func (o *KubeObject) SetName(name string) error {
	return o.SetNestedField(name, "metadata", "name")
}

func (o *KubeObject) GetNamespace() string {
	s, _, _ := o.obj.GetNestedString("metadata", "namespace")
	return s
}

// GetGKNNString returns with Group, Kind, Namespace, Name in a human-readable string
func (o *KubeObject) GetGKNNString() string {
	return fmt.Sprintf("%s/%s/%s", o.GroupKind().String(), o.GetNamespace(), o.GetName())
}

// IsNamespaceScoped tells whether a k8s resource is namespace scoped. If the KubeObject resource is a customized, it
// determines the namespace scope by checking whether `metadata.namespace` is set.
func (o *KubeObject) IsNamespaceScoped() bool {
	tm := yaml.TypeMeta{Kind: o.GetKind(), APIVersion: o.GetAPIVersion()}
	if nsScoped, ok := node.PrecomputedIsNamespaceScoped[tm]; ok {
		return nsScoped
	}
	// TODO(yuwenma): parse the resource openapi schema to know its scope status.
	return o.HasNamespace()
}

// IsClusterScoped tells whether a resource is cluster scoped.
func (o *KubeObject) IsClusterScoped() bool {
	return !o.IsNamespaceScoped()
}

func (o *KubeObject) HasNamespace() bool {
	_, found, _ := o.obj.GetNestedString("metadata", "namespace")
	return found
}

func (o *KubeObject) SetNamespace(name string) error {
	return o.SetNestedField(name, "metadata", "namespace")
}

func (o *KubeObject) SetAnnotation(k, v string) error {
	// Keep upstream-identifier untouched from users
	if k == UpstreamIdentifier {
		return ErrAttemptToTouchUpstreamIdentifier{}
	}
	if err := o.SetNestedField(v, "metadata", "annotations", k); err != nil {
		return fmt.Errorf("cannot set metadata annotations '%v': %v", k, err)
	}
	return nil
}

// GetAnnotations returns all annotations.
func (o *KubeObject) GetAnnotations() map[string]string {
	v, _, _ := o.obj.GetNestedStringMap("metadata", "annotations")
	if v == nil {
		return map[string]string{}
	}
	return v
}

// GetAnnotation returns one annotation with key k.
func (o *KubeObject) GetAnnotation(k string) string {
	v, _, _ := o.obj.GetNestedString("metadata", "annotations", k)
	return v
}
func (o *KubeObject) RemoveAnnotation(k string) error {
	_, err := o.RemoveNestedField("metadata", "annotations", k)
	return err
}

// HasAnnotations returns whether the KubeObject has all the given annotations.
func (o *KubeObject) HasAnnotations(annotations map[string]string) bool {
	kubeObjectLabels := o.GetAnnotations()
	for k, v := range annotations {
		kubeObjectValue, found := kubeObjectLabels[k]
		if !found || kubeObjectValue != v {
			return false
		}
	}
	return true
}

// RemoveAnnotationsIfEmpty removes the annotations field when it has zero annotations.
func (o *KubeObject) RemoveAnnotationsIfEmpty() error {
	annotations, found, err := o.obj.GetNestedStringMap("metadata", "annotations")
	if err != nil {
		return err
	}
	if found && len(annotations) == 0 {
		_, err = o.obj.RemoveNestedField("metadata", "annotations")
		return err
	}
	return nil
}
func (o *KubeObject) SetLabel(k, v string) error {
	return o.SetNestedField(v, "metadata", "labels", k)
}

// GetLabel returns one label with key k.
func (o *KubeObject) GetLabel(k string) string {
	v, _, _ := o.obj.GetNestedString("metadata", "labels", k)
	return v
}
func (o *KubeObject) RemoveLabel(k string) error {
	_, err := o.RemoveNestedField("metadata", "labels", k)
	return err
}

// GetLabels returns all labels.
func (o *KubeObject) GetLabels() map[string]string {
	v, _, _ := o.obj.GetNestedStringMap("metadata", "labels")
	if v == nil {
		return map[string]string{}
	}
	return v
}

// HasLabels returns whether the KubeObject has all the given labels
func (o *KubeObject) HasLabels(labels map[string]string) bool {
	kubeObjectLabels := o.GetLabels()
	for k, v := range labels {
		kubeObjectValue, found := kubeObjectLabels[k]
		if !found || kubeObjectValue != v {
			return false
		}
	}
	return true
}
func (o *KubeObject) PathAnnotation() string {
	return o.GetAnnotation(kioutil.PathAnnotation)
}
func (o *KubeObject) SetPathAnnotation(path string) error {
	return o.SetAnnotation(kioutil.PathAnnotation, path)
}

// IndexAnnotation returns -1 if not found.
func (o *KubeObject) IndexAnnotation() int {
	anno := o.GetAnnotation(kioutil.IndexAnnotation)
	if anno == "" {
		return -1
	}
	i, _ := strconv.Atoi(anno)
	return i
}
func (o *KubeObject) SetIndexAnnotation(index int) error {
	return o.SetAnnotation(kioutil.IndexAnnotation, strconv.Itoa(index))
}

// IDAnnotation return -1 if not found.
func (o *KubeObject) IDAnnotation() int {
	anno := o.GetAnnotation(kioutil.IdAnnotation)

	if anno == "" {
		return -1
	}
	i, _ := strconv.Atoi(anno)
	return i
}

// IsMetaResource returns a function that checks if a KubeObject is a meta resource. For now
// this just includes the Kptfile
func IsMetaResource() func(*KubeObject) bool {
	return IsGVK("kpt.dev", "v1", "Kptfile")
}
func (o *KubeObject) IsEmpty() bool {
	return o == nil || o.obj == nil || yaml.IsYNodeEmptyMap(o.obj.Node())
}
func NewEmptyKubeObject() *KubeObject {
	subObject := SubObject{parentGVK: schema.GroupVersionKind{}, obj: node.NewMap(nil), fieldpath: ""}
	return &KubeObject{subObject}
}
func asKubeObject(mapVariant *node.MapVariant) *KubeObject {
	group, _, _ := mapVariant.GetNestedString("group")
	version, _, _ := mapVariant.GetNestedString("version")
	kind, _, _ := mapVariant.GetNestedString("kind")
	gvk := schema.GroupVersionKind{Group: group, Version: version, Kind: kind}
	return &KubeObject{SubObject{parentGVK: gvk, obj: mapVariant, fieldpath: ""}}
}
func (o *KubeObject) node() *node.MapVariant {
	return o.obj
}
func rnodeToKubeObject(rn *yaml.RNode) *KubeObject {
	mapVariant := node.NewMap(rn.YNode())
	return asKubeObject(mapVariant)
}
func NewKubeObjectFromMap(m map[string]any) (*KubeObject, error) {
	rn, err := yaml.FromMap(m)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert unstructured/JSON map to KubeObject: %w", err)
	}
	return rnodeToKubeObject(rn), nil
}

// NewKubeObjectFromResourceNode creates a KubeObject from the deep copy of a yaml.RNode
func NewKubeObjectFromResourceNode(rn *yaml.RNode) *KubeObject {
	// create a deep copy of the RNode to avoid exposing internal state of the new KubeObject
	return rnodeToKubeObject(rn.Copy())
}

// CopyToResourceNode returns a deep copy of the KubeObject's internal yaml.RNode
func (o *KubeObject) CopyToResourceNode() *yaml.RNode {
	return yaml.NewRNode(o.obj.Node()).Copy()
}

// MoveToResourceNode transfers the ownership of the internal yaml nodes of the KubeObject
// into a new yaml.RNode, and leaves the original KubeObject empty.
func (o *KubeObject) MoveToResourceNode() *yaml.RNode {
	ynode := o.obj.Node()
	o.SubObject = NewEmptyKubeObject().SubObject
	return yaml.NewRNode(ynode)
}

// CopyToKubeObject makes a copy of the internal yaml nodes of the RNode into a new KubeObject.
func CopyToKubeObject(rn *yaml.RNode) *KubeObject {
	return rnodeToKubeObject(rn.Copy())
}

// MoveToKubeObject transfers the ownership of the internal yaml nodes of the RNode
// into a new KubeObject, and leaves the original RNode empty.
func MoveToKubeObject(rn *yaml.RNode) *KubeObject {
	ret := rnodeToKubeObject(rn)
	*rn = *yaml.MakeNullNode()
	return ret
}

// Copy returns a deep copy of the KubeObject
func (o *KubeObject) Copy() *KubeObject {
	ynode := yaml.CopyYNode(o.obj.Node())
	mapVariant := node.NewMap(ynode)
	return &KubeObject{SubObject{parentGVK: o.parentGVK, obj: mapVariant, fieldpath: ""}}
}
