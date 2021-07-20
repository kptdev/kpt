// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"fmt"

	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
)

// color is a type that captures the ANSI code for colors on the
// terminal.
type Color int

var (
	RED    Color = 31
	GREEN  Color = 32
	YELLOW Color = 33
)

// SprintfWithColor formats according to the provided pattern and returns
// the result as a string with the necessary ansii escape codes for
// color
func SprintfWithColor(color Color, format string, a ...interface{}) string {
	return fmt.Sprintf("%c[%dm", ESC, color) +
		fmt.Sprintf(format, a...) +
		fmt.Sprintf("%c[%dm", ESC, RESET)
}

// ColorForStatus returns the appropriate Color, which represents
// the ansii escape code, for different status values.
func ColorForStatus(s status.Status) (color Color, setColor bool) {
	switch s {
	case status.CurrentStatus:
		color = GREEN
		setColor = true
	case status.InProgressStatus:
		color = YELLOW
		setColor = true
	case status.FailedStatus:
		color = RED
		setColor = true
	}
	return
}
