// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/print/list"
)

func NewFormatter(ioStreams genericclioptions.IOStreams,
	previewStrategy common.DryRunStrategy) list.Formatter {
	return &formatter{
		ioStreams:       ioStreams,
		previewStrategy: previewStrategy,
	}
}

type formatter struct {
	previewStrategy common.DryRunStrategy
	ioStreams       genericclioptions.IOStreams
}

func (jf *formatter) FormatApplyEvent(ae event.ApplyEvent, as *list.ApplyStats, c list.Collector) error {
	switch ae.Type {
	case event.ApplyEventCompleted:
		if err := jf.printEvent("apply", "completed", map[string]interface{}{
			"count":           as.Sum(),
			"createdCount":    as.Created,
			"unchangedCount":  as.Unchanged,
			"configuredCount": as.Configured,
			"serverSideCount": as.ServersideApplied,
			"failedCount":     as.Failed,
		}); err != nil {
			return err
		}

		for id, se := range c.LatestStatus() {
			if err := jf.printResourceStatus(id, se); err != nil {
				return err
			}
		}
	case event.ApplyEventResourceUpdate:
		gk := ae.Identifier.GroupKind
		eventInfo := map[string]interface{}{
			"group":     gk.Group,
			"kind":      gk.Kind,
			"namespace": ae.Identifier.Namespace,
			"name":      ae.Identifier.Name,
			"operation": ae.Operation.String(),
		}
		if ae.Error != nil {
			eventInfo["error"] = ae.Error.Error()
		}

		return jf.printEvent("apply", "resourceApplied", eventInfo)
	}
	return nil
}

func (jf *formatter) FormatStatusEvent(se event.StatusEvent, _ list.Collector) error {
	if se.Type == event.StatusEventResourceUpdate {
		id := se.Resource.Identifier
		return jf.printResourceStatus(id, se)
	}
	return nil
}

func (jf *formatter) printResourceStatus(id object.ObjMetadata, se event.StatusEvent) error {
	return jf.printEvent("status", "resourceStatus",
		map[string]interface{}{
			"group":     id.GroupKind.Group,
			"kind":      id.GroupKind.Kind,
			"namespace": id.Namespace,
			"name":      id.Name,
			"status":    se.Resource.Status.String(),
			"message":   se.Resource.Message,
		})
}

func (jf *formatter) FormatPruneEvent(pe event.PruneEvent, ps *list.PruneStats) error {
	switch pe.Type {
	case event.PruneEventCompleted:
		return jf.printEvent("prune", "completed", map[string]interface{}{
			"pruned":  ps.Pruned,
			"skipped": ps.Skipped,
		})
	case event.PruneEventResourceUpdate:
		gk := pe.Identifier.GroupKind
		return jf.printEvent("prune", "resourcePruned", map[string]interface{}{
			"group":     gk.Group,
			"kind":      gk.Kind,
			"namespace": pe.Identifier.Namespace,
			"name":      pe.Identifier.Name,
			"operation": pe.Operation.String(),
		})
	case event.PruneEventFailed:
		gk := pe.Identifier.GroupKind
		return jf.printEvent("prune", "resourceFailed", map[string]interface{}{
			"group":     gk.Group,
			"kind":      gk.Kind,
			"namespace": pe.Identifier.Namespace,
			"name":      pe.Identifier.Name,
			"error":     pe.Error.Error(),
		})
	}
	return nil
}

func (jf *formatter) FormatDeleteEvent(de event.DeleteEvent, ds *list.DeleteStats) error {
	switch de.Type {
	case event.DeleteEventCompleted:
		return jf.printEvent("delete", "completed", map[string]interface{}{
			"deleted": ds.Deleted,
			"skipped": ds.Skipped,
		})
	case event.DeleteEventResourceUpdate:
		gk := de.Identifier.GroupKind
		return jf.printEvent("delete", "resourceDeleted", map[string]interface{}{
			"group":     gk.Group,
			"kind":      gk.Kind,
			"namespace": de.Identifier.Namespace,
			"name":      de.Identifier.Name,
			"operation": de.Operation.String(),
		})
	case event.DeleteEventFailed:
		gk := de.Identifier.GroupKind
		return jf.printEvent("delete", "resourceFailed", map[string]interface{}{
			"group":     gk.Group,
			"kind":      gk.Kind,
			"namespace": de.Identifier.Namespace,
			"name":      de.Identifier.Name,
			"error":     de.Error.Error(),
		})
	}
	return nil
}

func (jf *formatter) FormatErrorEvent(ee event.ErrorEvent) error {
	return jf.printEvent("error", "error", map[string]interface{}{
		"error": ee.Err.Error(),
	})
}

func (jf *formatter) printEvent(t, eventType string, content map[string]interface{}) error {
	m := make(map[string]interface{})
	m["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	m["type"] = t
	m["eventType"] = eventType
	for key, val := range content {
		m[key] = val
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(jf.ioStreams.Out, string(b)+"\n")
	return err
}
