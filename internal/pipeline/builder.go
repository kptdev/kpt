// Copyright 2020 Google LLC
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
//
// Package pipeline provides struct definitions for Pipeline and utility
// methods to read and write a pipeline resource.
package pipeline

// Builder will modify and build the Pipeline
type Builder struct {
	p *Pipeline
}

// Build returns the Pipeline built by this builder
func (b *Builder) Build() *Pipeline {
	return b.p
}

// SetName sets Pipeline name
func (b *Builder) SetName(n string) *Builder {
	b.p.Name = n
	return b
}

// SetKind sets Pipeline kind
func (b *Builder) SetKind(k string) *Builder {
	b.p.Kind = k
	return b
}

// SetAPIVersion sets Pipeline API version
func (b *Builder) SetAPIVersion(v string) *Builder {
	b.p.APIVersion = v
	return b
}

// AddSources appends the sources to Pipeline
func (b *Builder) AddSources(s ...string) *Builder {
	b.p.Sources = append(b.p.Sources, s...)
	return b
}

// SetSources replaces the sources in Pipeline by s
func (b *Builder) SetSources(s []string) *Builder {
	b.p.Sources = s
	return b
}

// AddGenerators appends the generators to Pipeline
func (b *Builder) AddGenerators(g ...Function) *Builder {
	addFunctions(&b.p.Generators, g)
	return b
}

// SetGenerators replaces the generators in Pipeline by g
func (b *Builder) SetGenerators(g []Function) *Builder {
	b.p.Generators = g
	return b
}

// AddTransformers appends the transformers to Pipeline
func (b *Builder) AddTransformers(t ...Function) *Builder {
	addFunctions(&b.p.Transformers, t)
	return b
}

// SetTransformers replaces the transformers in Pipeline by t
func (b *Builder) SetTransformers(t []Function) *Builder {
	b.p.Transformers = t
	return b
}

// AddValidators appends the validators to Pipeline
func (b *Builder) AddValidators(v ...Function) *Builder {
	addFunctions(&b.p.Validators, v)
	return b
}

// SetValidators replaces the validators in Pipeline by v
func (b *Builder) SetValidators(v []Function) *Builder {
	b.p.Validators = v
	return b
}

func addFunctions(orig *[]Function, new []Function) {
	*orig = append(*orig, new...)
}

// NewBuilder returns a builder with default Pipeline set
func NewBuilder() *Builder {
	return &Builder{p: DefaultPipeline()}
}

// NewBuilderWithPipeline eturns a builder with a pre-defined
// Pipeline
func NewBuilderWithPipeline(p *Pipeline) *Builder {
	return &Builder{p: p}
}
