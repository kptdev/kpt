// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package printers

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/cmd/printers/events"
	"sigs.k8s.io/cli-utils/cmd/printers/json"
	"sigs.k8s.io/cli-utils/cmd/printers/printer"
	"sigs.k8s.io/cli-utils/cmd/printers/table"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/print/list"
)

const (
	EventsPrinter = "events"
	TablePrinter  = "table"
	JSONPrinter   = "json"
)

func GetPrinter(printerType string, ioStreams genericclioptions.IOStreams) printer.Printer {
	switch printerType { //nolint:gocritic
	case TablePrinter:
		return &table.Printer{
			IOStreams: ioStreams,
		}
	case JSONPrinter:
		return &list.BaseListPrinter{
			FormatterFactory: func(previewStrategy common.DryRunStrategy) list.Formatter {
				return json.NewFormatter(ioStreams, previewStrategy)
			},
		}
	default:
		return events.NewPrinter(ioStreams)
	}
}

func SupportedPrinters() []string {
	return []string{EventsPrinter, TablePrinter, JSONPrinter}
}

func DefaultPrinter() string {
	return EventsPrinter
}

func ValidatePrinterType(printerType string) bool {
	for _, p := range SupportedPrinters() {
		if printerType == p {
			return true
		}
	}
	return false
}
