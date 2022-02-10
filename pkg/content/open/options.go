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

package open

import "context"

type options struct {
	ctx       context.Context
	providers []ContentProvider
}

func makeOptions(opts ...Option) options {
	opt := options{}
	Options(opts...)(&opt)
	return opt
}

// Option is a functional option for content operations.
type Option func(opt *options)

// Options is a convenience to combine several options into one.
func Options(opts ...Option) Option {
	return func(opt *options) {
		for _, option := range opts {
			if option != nil {
				option(opt)
			}
		}
	}
}

func WithContext(ctx context.Context) Option {
	return func(opt *options) {
		opt.ctx = ctx
	}
}

// WithProviders option adds support for additional Reference location
// types, or replaces the default Content strategy for built-in Reference types.
func WithProviders(openers ...ContentProvider) Option {
	return func(opt *options) {
		opt.providers = append(opt.providers, openers...)
	}
}
