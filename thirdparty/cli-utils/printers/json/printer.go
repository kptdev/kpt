// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/print/list"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/printers/printer"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/common"
)

func NewPrinter(ioStreams genericclioptions.IOStreams) printer.Printer {
	return &list.BaseListPrinter{
		FormatterFactory: func(previewStrategy common.DryRunStrategy) list.Formatter {
			return NewFormatter(ioStreams, previewStrategy)
		},
	}
}
