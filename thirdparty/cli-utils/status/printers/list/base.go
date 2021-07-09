package list

import (
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	pollevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/print/list"
)

// BaseListPrinter implements the Printer interface and outputs the resource
// status information as a list of events as they happen.
type BaseListPrinter struct {
	Formatter list.Formatter
}

// Print takes an event channel and outputs the status events on the channel
// until the channel is closed. The provided cancelFunc is consulted on
// every event and is responsible for stopping the poller when appropriate.
// This function will block.
func (ep *BaseListPrinter) Print(ch <-chan pollevent.Event, identifiers []object.ObjMetadata,
	cancelFunc collector.ObserverFunc) error {
	coll := collector.NewResourceStatusCollector(identifiers)
	// The actual work is done by the collector, which will invoke the
	// callback on every event. In the callback we print the status
	// information and call the cancelFunc which is responsible for
	// stopping the poller at the correct time.
	done := coll.ListenWithObserver(ch, collector.ObserverFunc(
		func(statusCollector *collector.ResourceStatusCollector, e pollevent.Event) {
			err := ep.printStatusEvent(e)
			if err != nil {
				panic(err)
			}
			cancelFunc(statusCollector, e)
		}),
	)
	// Block until the done channel is closed.
	<-done
	if o := coll.LatestObservation(); o.Error != nil {
		return o.Error
	}
	return nil
}

func (ep *BaseListPrinter) printStatusEvent(se pollevent.Event) error {
	switch se.EventType {
	case pollevent.ResourceUpdateEvent:
		id := se.Resource.Identifier
		return ep.Formatter.FormatStatusEvent(event.StatusEvent{
			Identifier:       id,
			Resource:         se.Resource.Resource,
			PollResourceInfo: se.Resource,
		})
	case pollevent.ErrorEvent:
		return ep.Formatter.FormatErrorEvent(event.ErrorEvent{
			Err: se.Error,
		})
	}
	return nil
}
