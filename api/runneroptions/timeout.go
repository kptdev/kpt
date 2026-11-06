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
	"fmt"
	"time"

	fnresultv1 "github.com/kptdev/kpt/api/fnresult/v1"
)

// TimeoutExitCode is the exit code reported for a function that timed out.
// It is the exit code of a process that was killed by a signal.
const TimeoutExitCode = -1

// timeoutError is the cause of a context that ended because a time limit
// has passed.
type timeoutError struct {
	timeout time.Duration
	// wholeRun is true for the time limit of the whole run, and false for the
	// time limit of a single function.
	wholeRun bool
}

func (e *timeoutError) Error() string {
	return fmt.Sprintf("timed out after %v", e.timeout)
}

// WithRunTimeout returns a copy of ctx that ends when timeout has passed. The
// functions that are run with the returned context share this time limit, so
// it limits the whole run and not each function. When timeout is zero, ctx is
// returned as it is.
func WithRunTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return ctx, func() {
			// Nothing to cancel: ctx is returned as it is.
		}
	}
	return context.WithTimeoutCause(ctx, timeout, &timeoutError{timeout: timeout, wholeRun: true})
}

// WithFnTimeout returns the context that a single function is run with. The
// function is limited by timeout. When timeout is zero, the function is
// limited by the time limit of the whole run if parent has one, and by
// defaultTimeout otherwise.
func WithFnTimeout(parent context.Context, timeout, defaultTimeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if timeout == 0 {
		if _, ok := parent.Deadline(); ok {
			return context.WithCancel(parent)
		}
		timeout = defaultTimeout
	}
	return context.WithTimeoutCause(parent, timeout, &timeoutError{timeout: timeout})
}

// ContextError returns the error for a function whose ctx has ended, or nil
// if ctx has not ended. err is the error from running the function, which is
// nil if only a child process of the function was killed. stderr is called
// only if ctx has ended. truncateOutput is copied to the returned ExecError.
func ContextError(ctx context.Context, err error, stderr func() string, truncateOutput bool) error {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return newTimeoutExecError(ctx, err, stderr(), truncateOutput)
	case ctx.Err() != nil && err == nil:
		return fmt.Errorf("function canceled: %w", context.Cause(ctx))
	}
	return nil
}

// newTimeoutExecError returns the error for a function that was killed
// because the time limit of ctx has passed.
func newTimeoutExecError(ctx context.Context, err error, stderr string, truncateOutput bool) *fnresultv1.ExecError {
	execErr := &fnresultv1.ExecError{
		OriginalErr:    err,
		ExitCode:       TimeoutExitCode,
		Stderr:         stderr,
		TruncateOutput: truncateOutput,
	}
	var cause *timeoutError
	if errors.As(context.Cause(ctx), &cause) {
		execErr.Timeout = cause.timeout
		execErr.RunTimeout = cause.wholeRun
	}
	return execErr
}
