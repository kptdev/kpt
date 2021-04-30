// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package printers

import (
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/status/printers/event"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/status/printers/printer"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/status/printers/table"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// CreatePrinter return an implementation of the Printer interface. The
// actual implementation is based on the printerType requested.
func CreatePrinter(printerType string, ioStreams genericclioptions.IOStreams) (printer.Printer, error) {
	switch printerType {
	case "table":
		return table.NewTablePrinter(ioStreams), nil
	default:
		return event.NewEventPrinter(ioStreams), nil
	}
}
