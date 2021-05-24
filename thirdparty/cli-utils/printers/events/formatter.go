// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/print/list"
)

func NewFormatter(ioStreams genericclioptions.IOStreams,
	dryRunStrategy common.DryRunStrategy) list.Formatter {
	return &formatter{
		print: getPrintFunc(ioStreams.Out, dryRunStrategy),
	}
}

type formatter struct {
	print printFunc
}

func (ef *formatter) FormatApplyEvent(ae event.ApplyEvent) error {
	gk := ae.Identifier.GroupKind
	name := ae.Identifier.Name
	if ae.Error != nil {
		ef.print("%s apply failed: %s", resourceIDToString(gk, name),
			ae.Error.Error())
	} else {
		ef.print("%s %s", resourceIDToString(gk, name),
			strings.ToLower(ae.Operation.String()))
	}
	return nil
}

func (ef *formatter) FormatStatusEvent(se event.StatusEvent) error {
	id := se.Identifier
	ef.printResourceStatus(id, se)
	return nil
}

func (ef *formatter) FormatPruneEvent(pe event.PruneEvent) error {
	gk := pe.Identifier.GroupKind
	if pe.Error != nil {
		ef.print("%s prune failed: %s", resourceIDToString(gk, pe.Identifier.Name),
			pe.Error.Error())
		return nil
	}

	switch pe.Operation {
	case event.Pruned:
		ef.print("%s pruned", resourceIDToString(gk, pe.Identifier.Name))
	case event.PruneSkipped:
		ef.print("%s prune skipped", resourceIDToString(gk, pe.Identifier.Name))
	}
	return nil
}

func (ef *formatter) FormatDeleteEvent(de event.DeleteEvent) error {
	gk := de.Identifier.GroupKind
	name := de.Identifier.Name

	if de.Error != nil {
		ef.print("%s deletion failed: %s", resourceIDToString(gk, name),
			de.Error.Error())
		return nil
	}

	switch de.Operation {
	case event.Deleted:
		ef.print("%s deleted", resourceIDToString(gk, name))
	case event.DeleteSkipped:
		ef.print("%s delete skipped", resourceIDToString(gk, name))
	}
	return nil
}

func (ef *formatter) FormatErrorEvent(_ event.ErrorEvent) error {
	return nil
}

func (ef *formatter) FormatActionGroupEvent(age event.ActionGroupEvent, ags []event.ActionGroup,
	as *list.ApplyStats, ps *list.PruneStats, ds *list.DeleteStats, c list.Collector) error {
	if age.Action == event.ApplyAction && age.Type == event.Finished {
		output := fmt.Sprintf("%d resource(s) applied. %d created, %d unchanged, %d configured, %d failed",
			as.Sum(), as.Created, as.Unchanged, as.Configured, as.Failed)
		// Only print information about serverside apply if some of the
		// resources actually were applied serverside.
		if as.ServersideApplied > 0 {
			output += fmt.Sprintf(", %d serverside applied", as.ServersideApplied)
		}
		ef.print(output)
	}

	if age.Action == event.PruneAction && age.Type == event.Finished {
		ef.print("%d resource(s) pruned, %d skipped, %d failed", ps.Pruned, ps.Skipped, ps.Failed)
	}

	if age.Action == event.DeleteAction && age.Type == event.Finished {
		ef.print("%d resource(s) deleted, %d skipped", ds.Deleted, ds.Skipped)
	}

	if age.Action == event.WaitAction && age.Type == event.Started {
		ag, found := list.ActionGroupByName(age.GroupName, ags)
		if !found {
			panic(fmt.Errorf("unknown action group name %q", age.GroupName))
		}
		for id, se := range c.LatestStatus() {
			// Only print information about objects that we actually care about
			// for this wait task.
			if found := object.ObjMetas(ag.Identifiers).Contains(id); found {
				ef.printResourceStatus(id, se)
			}
		}
	}
	return nil
}

func (ef *formatter) printResourceStatus(id object.ObjMetadata, se event.StatusEvent) {
	ef.print("%s is %s: %s", resourceIDToString(id.GroupKind, id.Name),
		se.PollResourceInfo.Status.String(), se.PollResourceInfo.Message)
}

// resourceIDToString returns the string representation of a GroupKind and a resource name.
func resourceIDToString(gk schema.GroupKind, name string) string {
	return fmt.Sprintf("%s/%s", strings.ToLower(gk.String()), name)
}

type printFunc func(format string, a ...interface{})

func getPrintFunc(w io.Writer, dryRunStrategy common.DryRunStrategy) printFunc {
	return func(format string, a ...interface{}) {
		if dryRunStrategy.ClientDryRun() {
			format += " (dry-run)"
		} else if dryRunStrategy.ServerDryRun() {
			format += " (dry-run-server)"
		}
		fmt.Fprintf(w, format+"\n", a...)
	}
}
