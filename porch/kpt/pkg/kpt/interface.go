// Copyright 2022 Google LLC
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

package kpt

import (
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

type Kpt interface {
	fn.Renderer
	fn.Evaluator

	// Evaluates kpt function on the resources comprising a (set of) package(s).
	// function is a reference to the kpt function (image URI).
	OldEval(input kio.Reader, function string, config kio.Reader, output kio.Writer) error

	// TODO: accept a function evaluation strategy interface to overwrite docker-based evaluation.
	OldRender(inupt kio.Reader, output kio.Writer) error
}
