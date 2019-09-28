// Copyright 2019 Google LLC
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

package sets

type Set map[string]interface{}

func (s Set) Len() int {
	return len(s)
}

func (s Set) List() []string {
	var val []string
	for k := range s {
		val = append(val, k)
	}
	return val
}

func (s Set) Insert(vals ...string) {
	for _, val := range vals {
		s[val] = nil
	}
}

func (s Set) Difference(s2 Set) Set {
	s3 := Set{}
	for k := range s {
		if _, found := s2[k]; !found {
			s3.Insert(k)
		}
	}
	return s3
}

func (s Set) SymmetricDifference(s2 Set) Set {
	s3 := Set{}
	for k := range s {
		if _, found := s2[k]; !found {
			s3.Insert(k)
		}
	}
	for k := range s2 {
		if _, found := s[k]; !found {
			s3.Insert(k)
		}
	}
	return s3
}

func (s Set) Intersection(s2 Set) Set {
	s3 := Set{}
	for k := range s {
		if _, found := s2[k]; found {
			s3.Insert(k)
		}
	}
	return s3
}
