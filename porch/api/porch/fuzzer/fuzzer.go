package fuzzer

import (
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
)

var Funcs = func(codecs runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{}
}
