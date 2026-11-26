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
	"fmt"
	"os/exec"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// runCmd runs cmd in a Windows Job Object. Closing or terminating the job
// stops the function and every process it has started, including processes
// that keep the function's output handles open after the function exits.
func runCmd(ctx context.Context, cmd *exec.Cmd) error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return fmt.Errorf("create function job object: %w", err)
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limits)),
		uint32(unsafe.Sizeof(limits)),
	); err != nil {
		_ = windows.CloseHandle(job)
		return fmt.Errorf("configure function job object: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = windows.CloseHandle(job)
		return err
	}

	process, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false,
		uint32(cmd.Process.Pid),
	)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("open function process: %w", err)
	}
	defer windows.CloseHandle(process)

	if err := windows.AssignProcessToJobObject(job, process); err != nil {
		_ = windows.CloseHandle(job)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("assign function process to job object: %w", err)
	}
	var jobMu sync.Mutex
	jobOpen := true
	closeJob := func() {
		jobMu.Lock()
		defer jobMu.Unlock()
		if jobOpen {
			_ = windows.CloseHandle(job)
			jobOpen = false
		}
	}
	terminateJob := func(exitCode uint32) {
		jobMu.Lock()
		defer jobMu.Unlock()
		if jobOpen {
			_ = windows.TerminateJobObject(job, exitCode)
		}
	}

	// Wait independently for the process so descendants are terminated as
	// soon as the function exits. This prevents cmd.Wait from being held open
	// by a child that inherited stdout or stderr.
	processDone := make(chan struct{})
	go func() {
		_, _ = windows.WaitForSingleObject(process, windows.INFINITE)
		closeJob()
		close(processDone)
	}()

	stopJob := context.AfterFunc(ctx, func() {
		terminateJob(1)
	})
	err = cmd.Wait()
	stopJob()
	<-processDone
	closeJob()
	return err
}
