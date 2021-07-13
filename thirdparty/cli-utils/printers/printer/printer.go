// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package printer

import (
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
)

type Printer interface {
	Print(ch <-chan event.Event, dryRunStrategy common.DryRunStrategy) error
}
