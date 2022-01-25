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

package kpt

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

type execKptOptions struct {
	Stdin []byte
}

// execKpt will execute kpt with the specified arguments
func execKpt(ctx context.Context, dir string, args []string, opt execKptOptions) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, "kpt", args...)

	var stdout, stderr bytes.Buffer
	if opt.Stdin != nil {
		cmd.Stdin = bytes.NewReader(opt.Stdin)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Dir = dir

	startTime := time.Now()
	if err := cmd.Run(); err != nil {
		klog.Warningf("running '%s' failed.\n  stdout:\n%s\n  stderr:\n%s", strings.Join(cmd.Args, " "), stdout.String(), stderr.String())
		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("failed to run kpt: %w", err)
	}

	klog.Infof("exec of '%s' succeeded in %v", strings.Join(cmd.Args, " "), time.Since(startTime))

	return stdout.Bytes(), stderr.Bytes(), nil
}
