// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package list

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	pollevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/print/list"
	"sigs.k8s.io/cli-utils/pkg/printers"
)

// BaseListPrinter implements the Printer interface and outputs the resource
// status information as a list of events as they happen.
type BaseListPrinter struct {
	Formatter list.Formatter
	Format    string
	Data      *PrintData
}

// PrintData records data required for printing
type PrintData struct {
	Identifiers object.ObjMetadataSet
	InvNameMap  map[object.ObjMetadata]string
	StatusSet   map[string]bool
}

// PrintError print out errors when received error events
func (ep *BaseListPrinter) PrintError(e error) error {
	err := ep.Formatter.FormatErrorEvent(event.ErrorEvent{Err: e})
	if err != nil {
		return err
	}
	return nil
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
	switch se.Type {
	case pollevent.ResourceUpdateEvent:
		id := se.Resource.Identifier
		var invName string
		var ok bool
		if invName, ok = ep.Data.InvNameMap[id]; !ok {
			return fmt.Errorf("Resource not found\n")
		}
		// filter out status that are not assigned
		statusString := se.Resource.Status.String()
		if _, ok := ep.Data.StatusSet[strings.ToLower(statusString)]; len(ep.Data.StatusSet) != 0 && !ok {
			return nil
		}
		switch ep.Format {
		case printers.EventsPrinter:
			_, err := fmt.Printf("%s/%s/%s/%s is %s: %s\n", invName,
				strings.ToLower(id.GroupKind.String()), id.Namespace, id.Name, statusString, se.Resource.Message)
			return err
		case printers.JSONPrinter:
			eventInfo := ep.createJsonObj(id)
			eventInfo["inventory-name"] = invName
			eventInfo["status"] = statusString
			eventInfo["message"] = se.Resource.Message
			b, err := json.Marshal(eventInfo)
			if err != nil {
				return err
			}
			_, err = fmt.Println(string(b))
			return err
		default:
			return fmt.Errorf("No such printer type\n")
		}
	case pollevent.ErrorEvent:
		return ep.Formatter.FormatErrorEvent(event.ErrorEvent{
			Err: se.Error,
		})
	}
	return nil
}

func (ep *BaseListPrinter) createJsonObj(id object.ObjMetadata) map[string]interface{} {
	return map[string]interface{}{
		"group":     id.GroupKind.Group,
		"kind":      id.GroupKind.Kind,
		"namespace": id.Namespace,
		"name":      id.Name,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"type":      "status",
	}
}
