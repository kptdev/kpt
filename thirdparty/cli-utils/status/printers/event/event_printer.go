// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package event

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	pollevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// eventPrinter implements the Printer interface and outputs the resource
// status information as a list of events as they happen.
type eventPrinter struct {
	ioStreams genericclioptions.IOStreams
}

// NewEventPrinter returns a new instance of the eventPrinter.
func NewEventPrinter(ioStreams genericclioptions.IOStreams) *eventPrinter {
	return &eventPrinter{
		ioStreams: ioStreams,
	}
}

// Print takes an event channel and outputs the status events on the channel
// until the channel is closed. The provided cancelFunc is consulted on
// every event and is responsible for stopping the poller when appropriate.
// This function will block.
func (ep *eventPrinter) Print(ch <-chan pollevent.Event, identifiers []object.ObjMetadata,
	cancelFunc collector.ObserverFunc) {
	coll := collector.NewResourceStatusCollector(identifiers)
	// The actual work is done by the collector, which will invoke the
	// callback on every event. In the callback we print the status
	// information and call the cancelFunc which is responsible for
	// stopping the poller at the correct time.
	done := coll.ListenWithObserver(ch, collector.ObserverFunc(
		func(statusCollector *collector.ResourceStatusCollector, e pollevent.Event) {
			ep.printStatusEvent(e)
			cancelFunc(statusCollector, e)
		}),
	)
	// Block until the done channel is closed.
	<-done
}

func (ep *eventPrinter) printStatusEvent(se pollevent.Event) {
	switch se.EventType {
	case pollevent.ResourceUpdateEvent:
		id := se.Resource.Identifier
		printResourceStatus(id, se, ep.ioStreams)
	case pollevent.ErrorEvent:
		id := se.Resource.Identifier
		gk := id.GroupKind
		fmt.Fprintf(ep.ioStreams.Out, "%s error: %s\n", resourceIDToString(gk, id.Name),
			se.Error.Error())
	}
}

// resourceIDToString returns the string representation of a GroupKind and a resource name.
func resourceIDToString(gk schema.GroupKind, name string) string {
	return fmt.Sprintf("%s/%s", strings.ToLower(gk.String()), name)
}

func printResourceStatus(id object.ObjMetadata, se pollevent.Event, ioStreams genericclioptions.IOStreams) {
	fmt.Fprintf(ioStreams.Out, "%s is %s: %s\n", resourceIDToString(id.GroupKind, id.Name),
		se.Resource.Status.String(), se.Resource.Message)
}
