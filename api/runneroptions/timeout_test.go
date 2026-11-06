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

package runneroptions

import (
	"context"
	"errors"
	"testing"
	"time"

	fnresultv1 "github.com/kptdev/kpt/api/fnresult/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextError(t *testing.T) {
	stderr := func() string { return "fn stderr" }
	runErr := errors.New("signal: killed")

	t.Run("not ended", func(t *testing.T) {
		ctx, cancel := WithFnTimeout(context.Background(), time.Hour, time.Hour)
		defer cancel()
		assert.NoError(t, ContextError(ctx, runErr, stderr, false))
	})

	t.Run("function timeout", func(t *testing.T) {
		ctx, cancel := WithFnTimeout(context.Background(), time.Nanosecond, time.Hour)
		defer cancel()
		<-ctx.Done()
		var execErr *fnresultv1.ExecError
		require.ErrorAs(t, ContextError(ctx, runErr, stderr, true), &execErr)
		assert.Equal(t, TimeoutExitCode, execErr.ExitCode)
		assert.Equal(t, "fn stderr", execErr.Stderr)
		assert.True(t, execErr.TruncateOutput)
		assert.Equal(t, "function execution timeout of 1ns exceeded", execErr.TimeoutMessage())
	})

	t.Run("run timeout limits a function with no timeout", func(t *testing.T) {
		runCtx, cancelRun := WithRunTimeout(context.Background(), time.Nanosecond)
		defer cancelRun()
		ctx, cancel := WithFnTimeout(runCtx, 0, time.Hour)
		defer cancel()
		<-ctx.Done()
		var execErr *fnresultv1.ExecError
		require.ErrorAs(t, ContextError(ctx, runErr, stderr, false), &execErr)
		assert.Equal(t, "pipeline execution timeout of 1ns exceeded", execErr.TimeoutMessage())
	})

	t.Run("canceled with no run error", func(t *testing.T) {
		ctx, cancel := WithFnTimeout(context.Background(), time.Hour, time.Hour)
		cancel()
		err := ContextError(ctx, nil, stderr, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}
