// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package printers

import (
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/status/printers/list"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/status/printers/printer"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/status/printers/table"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/printers"
	"sigs.k8s.io/cli-utils/pkg/printers/events"
	"sigs.k8s.io/cli-utils/pkg/printers/json"
)

// CreatePrinter return an implementation of the Printer interface. The
// actual implementation is based on the printerType requested.
func CreatePrinter(printerType string, ioStreams genericclioptions.IOStreams, printData *list.PrintData) (printer.Printer, error) {
	switch printerType {
	case printers.TablePrinter:
		return table.NewTablePrinter(ioStreams, printData), nil
	case printers.JSONPrinter:
		return &list.BaseListPrinter{
			Formatter: json.NewFormatter(ioStreams, common.DryRunNone),
			Data:      printData,
		}, nil
	default:
		return &list.BaseListPrinter{
			Formatter: events.NewFormatter(ioStreams, common.DryRunNone),
			Data:      printData,
		}, nil
	}
}
