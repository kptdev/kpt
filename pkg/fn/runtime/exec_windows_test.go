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

//go:build windows

package runtime

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCmdTimeoutStopsFunction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cmd := exec.Command("cmd.exe", "/c", "ping -n 30 127.0.0.1 > NUL")
	started := time.Now()
	err := runCmd(ctx, cmd)

	require.Error(t, err)
	assert.Less(t, time.Since(started), 5*time.Second)
}

func TestRunCmdReturnsCommandError(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "exit 7")
	err := runCmd(context.Background(), cmd)

	assert.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 7, exitErr.ExitCode())
}
