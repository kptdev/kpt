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

package util

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
)

// KubernetesName returns the passed id if it less than maxLen, otherwise
// a truncated version of id with a unique hash of length hashLen appended
// with length maxLen. maxLen must be at least 5 + hashLen, and hashLen
// must be at least 4.
func KubernetesName(id string, hashLen, maxLen int) string {
	if hashLen < 4 {
		hashLen = 4
	}
	if maxLen < hashLen+5 {
		maxLen = hashLen + 5
	}

	if len(id) <= maxLen {
		return id
	}

	hash := sha1.Sum([]byte(id))
	stubIdx := maxLen - hashLen - 1
	return fmt.Sprintf("%s-%s", id[:stubIdx], hex.EncodeToString(hash[:])[:hashLen])
}
