// Copyright 2021 Google LLC
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

package location

import (
	"errors"
	"strings"
)

func ParseReference(location string, opts ...Option) (Reference, error) {
	opt := makeOptions(opts...)

	for _, parser := range opt.parsers {
		ref, err := parser(location, opt)
		if err != nil {
			return nil, err
		}
		if ref != nil {
			return ref, nil
		}
	}

	return nil, errors.New("not implemented")
}

func startsWith(value string, prefix string) (string, bool) {
	if parts := strings.SplitN(value, prefix, 2); len(parts) == 2 && len(parts[0]) == 0 {
		return parts[1], true
	}
	return prefix, false
}
