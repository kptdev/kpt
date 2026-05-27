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

package live

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClusterClientFactoryWithContext_NilIsNormalized verifies nil ctx is
// normalized to Background().
func TestNewClusterClientFactoryWithContext_NilIsNormalized(t *testing.T) {
	//nolint:staticcheck // SA1012: deliberately passing nil to exercise the nil-safety guard.
	ccf := NewClusterClientFactoryWithContext(nil)
	require.NotNil(t, ccf, "NewClusterClientFactoryWithContext returned nil")
	require.NotNil(t, ccf.Ctx, "expected nil ctx to be normalized to Background(); got nil")
	// Background() never cancels; Done() returns a nil channel.
	require.Nil(t, ccf.Ctx.Done(), "expected Background()-equivalent ctx; Done() returned non-nil")
}

// TestNewClusterClientFactoryWithContext_PreservesRealCtx stores the caller
// ctx and preserves cancellation.
func TestNewClusterClientFactoryWithContext_PreservesRealCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ccf := NewClusterClientFactoryWithContext(ctx)
	require.Same(t, ctx, ccf.Ctx, "factory stored a different ctx than the one passed in")
	assert.NotNil(t, ccf.Ctx.Done(), "ctx should expose Done channel")
	cancel()
	select {
	case <-ccf.Ctx.Done():
	default:
		require.FailNow(t, "factory ctx did not observe cancellation of the source ctx")
	}
}

// TestNewClusterClientFactory_StructLiteralPathTolerated keeps legacy nil ctx.
func TestNewClusterClientFactory_StructLiteralPathTolerated(t *testing.T) {
	// Exercises the legacy constructor which leaves Ctx unset.
	ccf := NewClusterClientFactory()
	require.Nil(t, ccf.Ctx, "legacy constructor must not synthesize a ctx; callers rely on nil to signal opt-out")
	// We don't call ccf.NewClient here because it needs a real
	// cmdutil.Factory; the observable contract is that inside NewClient
	// the nil Ctx is normalized to Background(). That path is exercised
	// in the existing apply/destroy tests via the CLI integration tests.
}
