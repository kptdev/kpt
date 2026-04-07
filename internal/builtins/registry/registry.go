// Copyright 2026 The kpt Authors
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

package registry

import (
	"io"
	"strings"
	"sync"

	"k8s.io/klog/v2"
)

type BuiltinFunction interface {
	ImageName() string
	Run(r io.Reader, w io.Writer, stderr io.Writer) error
}

var (
	mu       sync.RWMutex
	registry = map[string]BuiltinFunction{}
)

func Register(fn BuiltinFunction) {
	mu.Lock()
	defer mu.Unlock()
	registry[normalizeImage(fn.ImageName())] = fn
}

func Lookup(imageName string) BuiltinFunction {
	mu.RLock()
	defer mu.RUnlock()
	normalized := normalizeImage(imageName)
	if strings.HasSuffix(imageName, ":latest") ||
		strings.Contains(imageName, "@sha256:") {
		return nil
	}
	fn := registry[normalized]
	if fn != nil && imageName != normalized {
		klog.Warningf("WARNING: builtin function %q is being used instead of the requested image %q. "+
			"The built-in implementation may differ from the pinned version.", normalized, imageName)
	}
	return fn
}

func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

func normalizeImage(image string) string {
	if idx := strings.Index(image, "@"); idx != -1 {
		image = image[:idx]
	}
	parts := strings.Split(image, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if idx := strings.Index(last, ":"); idx != -1 {
			parts[len(parts)-1] = last[:idx]
		}
	}
	return strings.Join(parts, "/")
}
