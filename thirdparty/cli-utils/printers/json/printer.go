// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/printers/printer"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/print/list"
)

func NewPrinter(ioStreams genericclioptions.IOStreams) printer.Printer {
	return &list.BaseListPrinter{
		IOStreams:        ioStreams,
		FormatterFactory: NewFormatter,
	}
}
