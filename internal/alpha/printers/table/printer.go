// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"fmt"
	"io"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	pollingevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
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
	coll := newResourceStateCollector(initEvent.ActionGroups, t.IOStreams.Out)

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
// TODO: We should have the number of columns and their widths be
// dependent on the space available.
var (
	unifiedStatusColumnDef = table.ColumnDef{
		// Column containing the overall progress.
		ColumnName:        "progress",
		ColumnHeader:      "PROGRESS",
		ColumnWidth:       80,
		PrintResourceFunc: printProgress,
	}

	alphaColumns = []table.ColumnDefinition{
		table.MustColumn("namespace"),
		table.MustColumn("resource"),

		// We are trying out a "single column" model here
		unifiedStatusColumnDef,
	}
)

// runPrintLoop starts a new goroutine that will regularly fetch the
// latest state from the collector and update the table.
func (t *Printer) runPrintLoop(coll *resourceStateCollector, stop chan struct{}) chan struct{} {
	finished := make(chan struct{})

	baseTablePrinter := table.BaseTablePrinter{
		IOStreams: t.IOStreams,
		Columns:   alphaColumns,
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

func printProgress(w io.Writer, width int, r table.Resource) (int, error) {
	var resInfo *resourceInfo
	switch res := r.(type) {
	case *resourceInfo:
		resInfo = res
	default:
		return 0, fmt.Errorf("unexpected type %T", r)
	}

	text, details, err := getProgress(resInfo)
	if err != nil {
		return 0, err
	}
	if details != "" {
		text += " " + details
	}

	if len(text) > width {
		text = text[:width]
	}
	n, err := fmt.Fprint(w, text)
	if err != nil {
		return n, err
	}
	return len(text), err
}

func getProgress(resInfo *resourceInfo) (string, string, error) {
	printStatus := false
	var text string
	var details string
	switch resInfo.ResourceAction {
	case event.ApplyAction:
		switch resInfo.lastApplyEvent.Status {
		case event.ApplyPending:
			text = "PendingApply"
		case event.ApplySuccessful:
			text = "Applied"
			printStatus = true
		case event.ApplySkipped:
			text = "SkippedApply"

		case event.ApplyFailed:
			text = "ApplyFailed"

		default:
			return "", "", fmt.Errorf("unknown ApplyStatus: %v", resInfo.lastApplyEvent.Status)
		}

		if resInfo.lastApplyEvent.Error != nil {
			details = fmt.Sprintf("error:%+v", resInfo.lastApplyEvent.Error)
		}

	case event.PruneAction:
		switch resInfo.lastPruneEvent.Status {
		case event.PrunePending:
			text = "PendingDeletion"
		case event.PruneSuccessful:
			text = "Deleted"
		case event.PruneSkipped:
			text = "DeletionSkipped"
		case event.PruneFailed:
			text = "DeletionFailed"
			text += fmt.Sprintf(" %+v", resInfo.lastPruneEvent.Error)

		default:
			return "", "", fmt.Errorf("unknown PruneStatus: %v", resInfo.lastPruneEvent.Status)
		}

		if resInfo.lastPruneEvent.Error != nil {
			details = fmt.Sprintf("error:%+v", resInfo.lastPruneEvent.Error)
		}

	default:
		return "", "", fmt.Errorf("unknown ResourceAction %v", resInfo.ResourceAction)
	}

	rs := resInfo.ResourceStatus()
	if printStatus && rs != nil {
		s := rs.Status.String()

		color, setColor := printcommon.ColorForStatus(rs.Status)
		if setColor {
			s = printcommon.SprintfWithColor(color, s)
		}

		text = s

		if resInfo.ResourceAction == event.WaitAction {
			text += " WaitStatus:" + resInfo.WaitStatus.String()
		}

		conditionStrings := getConditions(rs)
		if rs.Status != status.CurrentStatus {
			text += " Conditions:" + strings.Join(conditionStrings, ",")
		}

		var message string
		if rs.Error != nil {
			message = rs.Error.Error()
		} else {
			switch rs.Status {
			case status.CurrentStatus:
				// Don't print the message when things are OK
			default:
				message = rs.Message
			}
		}

		if message != "" {
			details += " message:" + message
		}

		// TODO: Need to wait for observedGeneration I think, as it is exiting before conditions are observed
	}

	return text, details, nil
}

func getConditions(rs *pollingevent.ResourceStatus) []string {
	u := rs.Resource
	if u == nil {
		return nil
	}

	// TODO: Should we be using kstatus here?

	conditions, found, err := unstructured.NestedSlice(u.Object,
		"status", "conditions")
	if !found || err != nil || len(conditions) == 0 {
		return nil
	}

	var conditionStrings []string
	for _, cond := range conditions {
		condition := cond.(map[string]interface{})
		conditionType := condition["type"].(string)
		conditionStatus := condition["status"].(string)
		conditionReason := condition["reason"].(string)
		lastTransitionTime := condition["lastTransitionTime"].(string)

		// TODO: Colors should be line based, pending should be light gray
		var color printcommon.Color
		switch conditionStatus {
		case "True":
			color = printcommon.GREEN
		case "False":
			color = printcommon.RED
		default:
			color = printcommon.YELLOW
		}

		text := conditionReason
		if text == "" {
			text = conditionType
		}

		if lastTransitionTime != "" && color != printcommon.GREEN {
			t, err := time.Parse(time.RFC3339, lastTransitionTime)
			if err != nil {
				klog.Warningf("failed to parse time %v: %v", lastTransitionTime, err)
			} else {
				text += " " + time.Since(t).Truncate(time.Second).String()
			}
		}

		s := printcommon.SprintfWithColor(color, text)
		conditionStrings = append(conditionStrings, s)
	}
	return conditionStrings
}
