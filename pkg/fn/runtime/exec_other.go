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

//go:build !unix

package runtime

import (
	"context"
	"os/exec"
	"time"
)

// execWaitDelay is how long kpt waits for the output of an exec function
// to be closed after the function has exited or has been killed.
const execWaitDelay = time.Second

// runCmd runs cmd and waits for it to finish. When the context of cmd ends,
// only the function is killed, not the processes that it started.
func runCmd(_ context.Context, cmd *exec.Cmd) error {
	// do not wait past the timeout for a child process of the function that
	// still has the output open
	cmd.WaitDelay = execWaitDelay
	return cmd.Run()
}
