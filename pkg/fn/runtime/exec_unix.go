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

//go:build unix

package runtime

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// forwardedSignals are the signals that kpt forwards to a running function.
// These are the signals that end kpt. A function runs in a process group of
// its own, so it does not receive them from the terminal together with kpt,
// e.g. on Ctrl-C.
var forwardedSignals = []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT}

// raiseSignal sends sig to kpt itself. It is a variable so that the tests can
// replace it.
var raiseSignal = func(sig syscall.Signal) {
	_ = syscall.Kill(syscall.Getpid(), sig)
}

// runCmd starts cmd in a process group of its own and waits for it to finish.
// The processes that the function starts are in the same group, so they are
// stopped together with the function:
//   - when ctx ends, the whole group is killed
//   - when kpt receives one of the forwardedSignals, the group receives it too
func runCmd(ctx context.Context, cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// listen before the function starts, so that no signal is missed
	signals := make(chan os.Signal, 1)
	notifyForwardedSignals(signals)
	defer signal.Stop(signals)

	if err := cmd.Start(); err != nil {
		return err
	}
	// the id of the process group is the process id of the function
	pgid := cmd.Process.Pid

	// When ctx ends, cmd kills the function, and only if it is still running.
	// A process that the function started can outlive the function and keep
	// its output open, so kill the group, also after the function has exited.
	stopKill := context.AfterFunc(ctx, func() {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	})
	defer stopKill()

	done := make(chan struct{})
	forwarded := make(chan struct{})
	go func() {
		defer close(forwarded)
		forwardSignals(signals, pgid, done)
	}()

	err := cmd.Wait()
	// A signal can arrive just as the function finishes. Stop listening first
	// and close done after that, so that forwardSignals still finds a signal
	// that is waiting. Do not return before it is forwarded: the signal is
	// about to end kpt, and kpt must not go on with the result of the
	// function first.
	signal.Stop(signals)
	close(done)
	<-forwarded
	return err
}

// notifyForwardedSignals makes c receive the forwardedSignals.
func notifyForwardedSignals(c chan<- os.Signal) {
	var signals []os.Signal
	for _, sig := range forwardedSignals {
		// keep a signal ignored if kpt was started that way, e.g. SIGHUP
		// under nohup
		if !signal.Ignored(sig) {
			signals = append(signals, sig)
		}
	}
	// Notify without signals would stand for all signals
	if len(signals) > 0 {
		signal.Notify(c, signals...)
	}
}

// forwardSignals forwards the signals that arrive on signals until done is
// closed. The caller stops the signals before it closes done, so a signal
// that is waiting at that point is the last one, and it is forwarded too.
func forwardSignals(signals chan os.Signal, pgid int, done <-chan struct{}) {
	for {
		select {
		case sig := <-signals:
			forwardSignal(signals, sig, pgid)
		case <-done:
			select {
			case sig := <-signals:
				forwardSignal(signals, sig, pgid)
			default:
			}
			return
		}
	}
}

// forwardSignal sends sig to the process group pgid, and then to kpt itself
// with the behavior that the signal has when no function is running. For the
// forwardedSignals that ends kpt.
func forwardSignal(signals chan os.Signal, sig os.Signal, pgid int) {
	s, ok := sig.(syscall.Signal)
	if !ok {
		return
	}
	_ = syscall.Kill(-pgid, s)
	// kpt must not receive the signal on signals again
	signal.Stop(signals)
	raiseSignal(s)
}
