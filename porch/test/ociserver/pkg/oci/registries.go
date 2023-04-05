// Copyright 2022 The kpt Authors
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

package oci

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
)

type Registries interface {
	FindRegistry(ctx context.Context, key string) (*Registry, error)
}

// StaticRegistries holds fixed registries
type StaticRegistries struct {
	mutex sync.Mutex
	repos map[string]*Registry
}

// NewStaticRegistries constructs an instance of StaticRegistries
func NewStaticRegistries() *StaticRegistries {
	return &StaticRegistries{
		repos: make(map[string]*Registry),
	}
}

// FindRegistry returns a registry registered under the specified id, or nil if none is registered.
func (r *StaticRegistries) FindRegistry(ctx context.Context, id string) (*Registry, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.repos[id], nil
}

// Add registers a git repository under the specified id
func (r *StaticRegistries) Add(id string, repo *Registry) error {
	if !isRegistryIDAllowed(id) {
		return fmt.Errorf("invalid name %q", id)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, found := r.repos[id]; found {
		return fmt.Errorf("repo %q already exists", id)
	}
	r.repos[id] = repo
	return nil
}

func NewDynamicRegistries(baseDir string, options []RegistryOption) *DynamicRegistries {
	return &DynamicRegistries{
		baseDir: baseDir,
		repos:   make(map[string]*dynamicRegistry),
		options: options,
	}
}

type DynamicRegistries struct {
	mutex   sync.Mutex
	repos   map[string]*dynamicRegistry
	baseDir string
	options []RegistryOption
}

type dynamicRegistry struct {
	mutex    sync.Mutex
	registry *Registry
	name     string
	dir      string
	options  []RegistryOption
}

func isRegistryIDAllowed(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			// OK
		} else if r >= '0' && r <= '9' {
			// OK
		} else {
			switch r {
			case '-':
				// OK
			case '/':
				// Allowed (!)
			default:
				return false
			}
		}
	}
	return true
}

func (r *DynamicRegistries) FindRegistry(ctx context.Context, id string) (*Registry, error) {
	dir := filepath.Join(r.baseDir, id)
	if !isRegistryIDAllowed(id) {
		return nil, fmt.Errorf("invalid name %q", id)
	}

	r.mutex.Lock()
	repo := r.repos[id]
	if repo == nil {
		repo = &dynamicRegistry{
			name:    id,
			dir:     dir,
			options: r.options,
		}
		r.repos[id] = repo
	}
	r.mutex.Unlock()

	return repo.open()
}

func (r *dynamicRegistry) open() (*Registry, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.registry == nil {
		baseDir := r.dir

		registry, err := NewRegistry(r.name, baseDir, r.options...)
		if err != nil {
			return nil, err
		}
		r.registry = registry
	}

	return r.registry, nil
}
