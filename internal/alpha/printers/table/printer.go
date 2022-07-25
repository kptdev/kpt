// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"fmt"
	"io"
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	printcommon "sigs.k8s.io/cli-utils/pkg/print/common"
	"sigs.k8s.io/cli-utils/pkg/print/table"
)

type Printer struct {
	IOStreams genericclioptions.IOStreams
}

func (t *Printer) Print(ch <-chan event.Event, _ common.DryRunStrategy, _ bool) error {
	// Wait for the init event that will give us the set of
	// resources.
	var initEvent event.InitEvent
	for e := range ch {
		if e.Type == event.InitType {
			initEvent = e.InitEvent
			break
		}
		// If we get an error event, we just print it and
		// exit. The error event signals a fatal error.
		if e.Type == event.ErrorType {
			return e.ErrorEvent.Err
		}
	}
	// Create a new collector and initialize it with the resources
	// we are interested in.
	coll := newResourceStateCollector(initEvent.ActionGroups)

	stop := make(chan struct{})

	// Start the goroutine that is responsible for
	// printing the latest state on a regular cadence.
	printCompleted := t.runPrintLoop(coll, stop)

	// Make the collector start listening on the eventChannel.
	done := coll.Listen(ch)

	// Block until all the collector has shut down. This means the
	// eventChannel has been closed and all events have been processed.
	var err error
	for msg := range done {
		err = msg.err
	}

	// Close the stop channel to notify the print goroutine that it should
	// shut down.
	close(stop)

	// Wait until the printCompleted channel is closed. This means
	// the printer has updated the UI with the latest state and
	// exited from the goroutine.
	<-printCompleted

	if err != nil {
		return err
	}
	// If no fatal errors happened, we will return a ResultError if
	// one or more resources failed to apply/prune or reconcile.
	return printcommon.ResultErrorFromStats(coll.stats)
}

// columns defines the columns we want to print
//TODO: We should have the number of columns and their widths be
// dependent on the space available.
var (
	actionColumnDef = table.ColumnDef{
		// Column containing the resource type and name. Currently it does not
		// print group or version since those are rarely needed to uniquely
		// distinguish two resources from each other. Just name and kind should
		// be enough in almost all cases and saves space in the output.
		ColumnName:   "action",
		ColumnHeader: "ACTION",
		ColumnWidth:  12,
		PrintResourceFunc: func(w io.Writer, width int, r table.Resource) (int,
			error) {
			var resInfo *resourceInfo
			switch res := r.(type) {
			case *resourceInfo:
				resInfo = res
			default:
				return 0, nil
			}

			var text string
			switch resInfo.ResourceAction {
			case event.ApplyAction:
				if resInfo.ApplyStatus != event.ApplyFailed {
					text = resInfo.ApplyStatus.String()
				}
			case event.PruneAction:
				if resInfo.PruneStatus != event.PruneFailed {
					text = resInfo.PruneStatus.String()
				}
			}

			if len(text) > width {
				text = text[:width]
			}
			_, err := fmt.Fprint(w, text)
			return len(text), err
		},
	}

	reconciledColumnDef = table.ColumnDef{
		// Column containing the reconciliation status.
		ColumnName:   "reconciled",
		ColumnHeader: "RECONCILED",
		ColumnWidth:  10,
		PrintResourceFunc: func(w io.Writer, width int, r table.Resource) (
			int,
			error,
		) {
			var resInfo *resourceInfo
			switch res := r.(type) {
			case *resourceInfo:
				resInfo = res
			default:
				return 0, nil
			}

			var text string
			switch resInfo.ResourceAction {
			case event.WaitAction:
				text = resInfo.WaitStatus.String()
			}

			if len(text) > width {
				text = text[:width]
			}
			_, err := fmt.Fprint(w, text)
			return len(text), err
		},
	}

	columns = []table.ColumnDefinition{
		table.MustColumn("namespace"),
		table.MustColumn("resource"),
		actionColumnDef,
		table.MustColumn("status"),
		reconciledColumnDef,
		table.MustColumn("conditions"),
		table.MustColumn("age"),
		table.MustColumn("message"),
	}
)

// runPrintLoop starts a new goroutine that will regularly fetch the
// latest state from the collector and update the table.
func (t *Printer) runPrintLoop(coll *resourceStateCollector, stop chan struct{}) chan struct{} {
	finished := make(chan struct{})

	baseTablePrinter := table.BaseTablePrinter{
		IOStreams: t.IOStreams,
		Columns:   columns,
	}

	linesPrinted := baseTablePrinter.PrintTable(coll.LatestState(), 0)

	go func() {
		defer close(finished)
		ticker := time.NewTicker(500 * time.Millisecond)
		for {
			select {
			case <-stop:
				ticker.Stop()
				latestState := coll.LatestState()
				linesPrinted = baseTablePrinter.PrintTable(latestState, linesPrinted)
				_, _ = fmt.Fprint(t.IOStreams.Out, "\n")
				return
			case <-ticker.C:
				latestState := coll.LatestState()
				linesPrinted = baseTablePrinter.PrintTable(latestState, linesPrinted)
			}
		}
	}()
	return finished
}
