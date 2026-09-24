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
	"strings"

	"github.com/kptdev/kpt/api/kubeobject/node"
	schema "github.com/kptdev/kpt/api/schema/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// SubObject represents a map within a KubeObject.
type SubObject struct {
	parentGVK schema.GroupVersionKind
	fieldpath string
	obj       *node.MapVariant
}

// NestedBool returns the bool value, if the field exist and a potential error.
func (o *SubObject) NestedBool(fields ...string) (bool, bool, error) {
	b, found, err := o.obj.GetNestedBool(fields...)
	if err != nil {
		var val bool
		return val, found, NewErrUnmatchedField(*o, fields, val)
	}
	return b, found, nil
}

// NestedString returns the string value, if the field exist and a potential error.
func (o *SubObject) NestedString(fields ...string) (string, bool, error) {
	s, found, err := o.obj.GetNestedString(fields...)
	if err != nil {
		var val string
		return val, found, NewErrUnmatchedField(*o, fields, val)
	}
	return s, found, nil
}

// NestedFloat64 returns the float64 value, if the field exist and a potential error.
func (o *SubObject) NestedFloat64(fields ...string) (float64, bool, error) {
	f, found, err := o.obj.GetNestedFloat(fields...)
	if err != nil {
		var val float64
		return val, found, NewErrUnmatchedField(*o, fields, val)
	}
	return f, found, nil
}

// NestedInt64 returns the int64 value, if the field exist and a potential error.
func (o *SubObject) NestedInt64(fields ...string) (int64, bool, error) {
	i, found, err := o.obj.GetNestedInt(fields...)
	if err != nil {
		var val int64
		return val, found, NewErrUnmatchedField(*o, fields, val)
	}
	return int64(i), found, nil
}

// NestedInt returns the int64 value, if the field exist and a potential error.
func (o *SubObject) NestedInt(fields ...string) (int, bool, error) {
	i, found, err := o.obj.GetNestedInt(fields...)
	if err != nil {
		var val int
		return val, found, NewErrUnmatchedField(*o, fields, val)
	}
	return i, found, nil
}

// NestedSlice accepts a slice of `fields` which represents the path to the slice component and
// return a slice of SubObjects as the first return value; whether the component exists or
// not as the second return value, and errors as the third return value.
func (o *SubObject) NestedSlice(fields ...string) (SliceSubObjects, bool, error) {
	// Expect a struct like SubObject.
	var obj struct{}
	var mapVariant *node.MapVariant
	if len(fields) > 1 {
		m, found, err := o.obj.GetNestedMap(fields[:len(fields)-1]...)
		if err != nil {
			return nil, found, NewErrUnmatchedField(*o, fields, obj)
		}
		if !found {
			return nil, found, nil
		}
		mapVariant = m
	} else {
		mapVariant = o.obj
	}
	sliceVal, found, err := mapVariant.GetNestedSlice(fields[len(fields)-1])
	if err != nil {
		return nil, found, NewErrUnmatchedField(*o, fields, obj)
	}
	if !found {
		return nil, found, nil
	}
	objects, err := sliceVal.Elements()
	if err != nil {
		return nil, found, err
	}
	var val []*SubObject
	for _, obj := range objects {
		val = append(val, &SubObject{obj: obj})
	}
	return val, true, nil
}

// NestedSubObject returns with a SubObject representing the YAML subtree under the path specified by `fields`
func (o *SubObject) NestedSubObject(fields ...string) (SubObject, bool, error) {
	var variant SubObject
	m, found, err := o.obj.GetNestedMap(fields...)
	if err != nil {
		return variant, found, NewErrUnmatchedField(*o, fields, variant)
	}
	if !found {
		return variant, found, nil
	}

	var rn yaml.RNode
	rn.SetYNode(m.Node())
	variant.obj = node.NewMap(rn.YNode())
	variant.parentGVK = o.parentGVK
	variant.fieldpath = o.fieldpath + "." + strings.Join(fields, ".")
	return variant, true, nil
}

// NestedResource returns a map[string]string value of a nested field, false if not found and an error if not a map[string]string type.
func (o *SubObject) NestedResource(ptr any, fields ...string) (bool, error) {
	if ptr == nil || reflect.ValueOf(ptr).Kind() != reflect.Pointer {
		return false, fmt.Errorf("ptr must be a pointer to an object")
	}
	k := reflect.TypeOf(ptr).Elem().Kind()
	if k != reflect.Struct && k != reflect.Map {
		return false, fmt.Errorf("expect struct or map, got %T", ptr)
	}
	m, found, err := o.obj.GetNestedMap(fields...)
	if err != nil {
		o.fieldpath = o.fieldpath + "." + strings.Join(fields, ".")
		return found, NewErrUnmatchedField(*o, fields, ptr)
	}
	if !found {
		return found, nil
	}
	err = m.Node().Decode(ptr)
	return true, err
}

// NestedStringMap returns a map[string]string value of a nested field, false if not found and an error if not a map[string]string type.
func (o *SubObject) NestedStringMap(fields ...string) (map[string]string, bool, error) {
	var variant map[string]string
	m, found, err := o.obj.GetNestedMap(fields...)
	if err != nil {
		return variant, found, NewErrUnmatchedField(*o, fields, variant)
	}
	if !found {
		return variant, false, nil
	}
	err = m.Node().Decode(&variant)
	return variant, found, err
}

// NestedStringSlice returns a []string value of a nested field, false if not found and an error if not a []string type.
func (o *SubObject) NestedStringSlice(fields ...string) ([]string, bool, error) {
	var variant []string
	s, found, err := o.obj.GetNestedSlice(fields...)
	if err != nil {
		return variant, found, NewErrUnmatchedField(*o, fields, variant)
	}
	if !found {
		return variant, false, nil
	}
	err = s.Node().Decode(&variant)
	return variant, true, err
}

// RemoveNestedField removes the field located by fields if found. It returns if the field
// is found and a potential error.
func (o *SubObject) RemoveNestedField(fields ...string) (bool, error) {
	found, err := func() (bool, error) {
		if o == nil {
			return false, fmt.Errorf("the object doesn't exist")
		}
		return o.obj.RemoveNestedField(fields...)
	}()
	if err != nil {
		return found, fmt.Errorf("unable to remove fields %v with error: %w", fields, err)
	}
	return found, nil
}

// onLockedFields locks the SubObject fields which are expected for kpt internal use only.
func (o *SubObject) onLockedFields(val any, fields ...string) error {
	if o.hasUpstreamIdentifier(val, fields...) {
		return ErrAttemptToTouchUpstreamIdentifier{}
	}
	return nil
}

// SetNestedField sets a nested field located by fields to the value provided as val. val
// should not be a yaml.RNode. If you want to deal with yaml.RNode, you should
// use Get method and modify the underlying yaml.Node.
func (o *SubObject) SetNestedField(val any, fields ...string) error {
	if err := o.onLockedFields(val, fields...); err != nil {
		return err
	}
	err := func() error {
		if val == nil {
			return fmt.Errorf("the passed-in object must not be nil")
		}
		if o == nil {
			return fmt.Errorf("the object doesn't exist")
		}
		if o.obj == nil {
			o.obj = node.NewMap(nil)
		}
		kind := reflect.ValueOf(val).Kind()
		if kind == reflect.Pointer {
			kind = reflect.TypeOf(val).Elem().Kind()
		}

		switch kind {
		case reflect.Struct, reflect.Map:
			m, err := node.TypedObjectToMapVariant(val)
			if err != nil {
				return err
			}
			return o.obj.SetNestedMap(m, fields...)
		case reflect.Slice:
			s, err := node.TypedObjectToSliceVariant(val)
			if err != nil {
				return err
			}
			return o.obj.SetNestedSlice(s, fields...)
		case reflect.String:
			var s string
			switch val := val.(type) {
			case string:
				s = val
			case *string:
				s = *val
			}
			return o.obj.SetNestedString(s, fields...)
		case reflect.Int, reflect.Int64:
			var i int
			switch val := val.(type) {
			case int:
				i = val
			case *int:
				i = *val
			case int64:
				i = int(val)
			case *int64:
				i = int(*val)
			}
			return o.obj.SetNestedInt(i, fields...)
		case reflect.Float64:
			var f float64
			switch val := val.(type) {
			case float64:
				f = val
			case *float64:
				f = *val
			}
			return o.obj.SetNestedFloat(f, fields...)
		case reflect.Bool:
			var b bool
			switch val := val.(type) {
			case bool:
				b = val
			case *bool:
				b = *val
			}
			return o.obj.SetNestedBool(b, fields...)
		default:
			return fmt.Errorf("unhandled kind %s", kind)
		}
	}()
	if err != nil {
		return fmt.Errorf("unable to set %v at fields %v with error: %w", val, fields, err)
	}
	return nil
}

// SetNestedInt sets the `fields` value to int `value`. It returns error if the fields type is not int.
func (o *SubObject) SetNestedInt(value int, fields ...string) error {
	return o.SetNestedField(value, fields...)
}

// SetNestedBool sets the `fields` value to bool `value`. It returns error if the fields type is not bool.
func (o *SubObject) SetNestedBool(value bool, fields ...string) error {
	return o.SetNestedField(value, fields...)
}

// SetNestedString sets the `fields` value to string `value`. It returns error if the fields type is not string.
func (o *SubObject) SetNestedString(value string, fields ...string) error {
	return o.SetNestedField(value, fields...)
}

// SetNestedStringMap sets the `fields` value to map[string]string `value`. It returns error if the fields type is not map[string]string.
func (o *SubObject) SetNestedStringMap(value map[string]string, fields ...string) error {
	return o.SetNestedField(value, fields...)
}

// UpdateNestedStringMap updates the map[string]string `fields` value with the (key, value) pairs in `values`.
// It returns error if the fields type is not map[string]string.
func (o *SubObject) UpdateNestedStringMap(values map[string]string, fields ...string) error {
	for field, value := range values {
		path := append(fields, field) //nolint:gocritic
		if err := o.SetNestedString(value, path...); err != nil {
			return fmt.Errorf("couldn't update field %s: %w", strings.Join(fields, "."), err)
		}
	}
	return nil
}

// SetNestedStringSlice sets the `fields` value to []string `value`. It returns error if the fields type is not []string.
func (o *SubObject) SetNestedStringSlice(value []string, fields ...string) error {
	return o.SetNestedField(value, fields...)
}

// As converts a KubeObject to the desired typed object. ptr must be
// a pointer to a typed object.
func (o *SubObject) As(ptr any) error {
	err := func() error {
		if o == nil {
			return fmt.Errorf("the object doesn't exist")
		}
		if ptr == nil || reflect.ValueOf(ptr).Kind() != reflect.Pointer {
			return fmt.Errorf("ptr must be a pointer to an object")
		}
		return node.MapVariantToTypedObject(o.obj, ptr)
	}()
	if err != nil {
		return fmt.Errorf("unable to convert object to %T with error: %w", ptr, err)
	}
	return nil
}

// Bytes serializes the object in yaml format.
func (o *SubObject) Bytes() []byte {
	doc := node.NewDoc([]*yaml.Node{o.obj.Node()}...)
	s, _ := doc.ToYAML()
	return s
}

// String serializes the object in yaml format.
func (o *SubObject) String() string {
	return string(o.Bytes())
}

func (o *SubObject) IsEmpty() bool {
	return o == nil || o.obj.IsEmpty()
}
func (o *SubObject) HasField(key string) bool {
	return o.obj.HasKey(key)
}
func (o *SubObject) UpsertMap(k string) *SubObject {
	m := o.obj.UpsertMap(k)
	return &SubObject{obj: m, parentGVK: o.parentGVK, fieldpath: o.fieldpath + "." + k}
}

// Set ensures that the value of `o` (this object) is the same as `newValue`,
// while keeps the formatting of the original object.
func (o *SubObject) Set(newValue *SubObject) error {
	o.obj.Set(newValue.obj)
	return nil
}

// SetFromTypedObject ensures that the value of `o` (this object) is the same as `newValue`,
// while keeps the formatting of the original object.
// `newValue` must be of type struct or map[string]...
func (o *SubObject) SetFromTypedObject(newValue any) error {
	kind := reflect.ValueOf(newValue).Kind()
	if kind == reflect.Pointer {
		kind = reflect.TypeOf(newValue).Elem().Kind()
	}
	if kind != reflect.Struct && kind != reflect.Map {
		return fmt.Errorf("expected struct or map, got %T", newValue)
	}

	newMap, err := node.TypedObjectToMapVariant(newValue)
	if err != nil {
		return err
	}
	o.obj.Set(newMap)
	return nil
}

// SetMap  accepts a single key `k`, and ensures that the value of `k` is the same as the map it received
// via `mapObject` in the form of a SubObject pointer.
func (o *SubObject) SetMap(mapObj *SubObject, k string) error {
	return o.obj.SetNestedMap(mapObj.obj, k)
}

// GetMap accepts a single key `k` whose value is expected to be a map. It returns
// the map in the form of a SubObject pointer.
// It panics with ErrSubObjectFields error if the field cannot be represented as a SubObject.
func (o *SubObject) GetMap(k string) *SubObject {
	var rn yaml.RNode
	val, found, err := o.obj.GetNestedValue(k)
	if err != nil || !found {
		return nil
	}
	rn.SetYNode(val.Node())
	return &SubObject{obj: node.NewMap(rn.YNode()), parentGVK: o.parentGVK, fieldpath: o.fieldpath + "." + k}
}

// GetBool accepts a single key `k` whose value is expected to be a boolean. It returns
// the int value of the `k`. It panics with errSubObjectFields error if the
// field is not an integer type.
func (o *SubObject) GetBool(k string) bool {
	val, _, _ := o.NestedBool(k)
	return val
}

// GetInt accepts a single key `k` whose value is expected to be an integer. It returns
// the int value of the `k`. It panics with errSubObjectFields error if the
// field is not an integer type.
func (o *SubObject) GetInt(k string) int64 {
	val, _, _ := o.NestedInt64(k)
	return val
}

// GetString accepts a single key `k` whose value is expected to be a string. It returns
// the value of the `k`. It panics with errSubObjectFields error if the
// field is not a string type.
func (o *SubObject) GetString(k string) string {
	val, _, _ := o.NestedString(k)
	return val
}

// GetSlice accepts a single key `k` whose value is expected to be a slice. It returns
// the value as a slice of SubObject. It panics with errSubObjectFields error if the
// field is not a slice type.
func (o *SubObject) GetSlice(k string) SliceSubObjects {
	val, _, _ := o.NestedSlice(k)
	return val
}

// SetSlice sets the SliceSubObjects to the given field. It creates the field if not exists.
// It returns an error if the field exists but not a slice type.
func (o *SubObject) SetSlice(objects SliceSubObjects, field string) error {
	s := node.NewSliceVariant()
	for _, element := range objects {
		s.Add(element.obj)
	}
	return o.obj.SetNestedSlice(s, field)
}

type SliceSubObjects []*SubObject

// MarshalJSON provides the custom encoding format for encode.json. This is used
// when KubeObject `Set` a slice of SubObjects.
func (s *SliceSubObjects) MarshalJSON() ([]byte, error) {
	seq := &yaml.Node{Kind: yaml.SequenceNode}
	for _, subObject := range *s {
		seq.Content = append(seq.Content, subObject.obj.Node())
	}
	return yaml.NewRNode(seq).MarshalJSON()
}
