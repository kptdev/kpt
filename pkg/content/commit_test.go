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

package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommitNotSupported(t *testing.T) {
	// arrange
	c := &testContent{}

	// act
	ref, lock, err := Commit(c)

	// assert
	assert.Nil(t, ref)
	assert.Nil(t, lock)
	assert.ErrorIs(t, err, ErrNotCommittable)
	assert.Equal(t, "invalid *content.testContent operation: not committable", err.Error())
}

type testContent struct {
}

func (testContent) Close() error {
	return nil
}

var _ Content = &testContent{}
