package search

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// visitor is implemented by structs which need to walk the configuration.
// visitor is provided to accept to walk configuration
type visitor interface {
	// visitScalar is called for each scalar field value on a resource
	// node is the scalar field value
	// path is the path to the field; path elements are separated by '.'
	// oa is the OpenAPI schema for the field
	visitScalar(node *yaml.RNode, path string) error

	// visitSequence is called for each sequence field value on a resource
	// node is the sequence field value
	// path is the path to the field
	// oa is the OpenAPI schema for the field
	visitSequence(node *yaml.RNode, path string) error

	// visitMapping is called for each Mapping field value on a resource
	// node is the mapping field value
	// path is the path to the field
	// oa is the OpenAPI schema for the field
	visitMapping(node *yaml.RNode, path string) error
}

// accept invokes the appropriate function on v for each field in object
func accept(v visitor, object *yaml.RNode) error {
	// get the OpenAPI for the type if it exists
	return acceptImpl(v, object, "")
}

// acceptImpl implements accept using recursion
func acceptImpl(v visitor, object *yaml.RNode, p string) error {
	switch object.YNode().Kind {
	case yaml.DocumentNode:
		// Traverse the child of the document
		return accept(v, yaml.NewRNode(object.YNode()))
	case yaml.MappingNode:
		if err := v.visitMapping(object, p); err != nil {
			return err
		}
		return object.VisitFields(func(node *yaml.MapNode) error {
			// get the schema for the field and propagate it
			// Traverse each field value
			return acceptImpl(v, node.Value, p+"."+node.Key.YNode().Value)
		})
	case yaml.SequenceNode:
		// get the schema for the sequence node, use the schema provided if not present
		// on the field
		if err := v.visitSequence(object, p); err != nil {
			return err
		}
		// get the schema for the elements
		return object.VisitElements(func(node *yaml.RNode) error {
			// Traverse each list element
			return acceptImpl(v, node, p)
		})
	case yaml.ScalarNode:
		// Visit the scalar field
		return v.visitScalar(object, p)
	}
	return nil
}
