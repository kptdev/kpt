// Copyright 2019 The kpt Authors
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

package stack

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
)

// New returns a new stack for elements of string type.
func New() *Stack {
	return &Stack{}
}

type Stack struct {
	slice []string
}

func (s *Stack) Push(str string) {
	s.slice = append(s.slice, str)
}

func (s *Stack) Pop() string {
	l := len(s.slice)
	if l == 0 {
		panic(fmt.Errorf("can't pop an empty stack"))
	}
	str := s.slice[l-1]
	s.slice = s.slice[:l-1]
	return str
}

func (s *Stack) Len() int {
	return len(s.slice)
}

// NewPkgStack returns a new stack for elements of *pkg.Pkg type.
func NewPkgStack() *PkgStack {
	return &PkgStack{}
}

type PkgStack struct {
	slice []*pkg.Pkg
}

func (ps *PkgStack) Push(p *pkg.Pkg) {
	ps.slice = append(ps.slice, p)
}

func (ps *PkgStack) PushAll(pkgs []*pkg.Pkg) {
	for i := range pkgs {
		p := pkgs[i]
		ps.Push(p)
	}
}

func (ps *PkgStack) Pop() *pkg.Pkg {
	l := len(ps.slice)
	if l == 0 {
		panic(fmt.Errorf("can't pop an empty stack"))
	}
	p := ps.slice[l-1]
	ps.slice = ps.slice[:l-1]
	return p
}

func (ps *PkgStack) Len() int {
	return len(ps.slice)
}
