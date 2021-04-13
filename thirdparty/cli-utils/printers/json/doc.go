// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

// Package json provides a printer that outputs the eventstream in json
// format. Each event is printed as a json object, so the output will
// appear as a stream of json objects, each representing a single event.
//
// Every event will contain the following properties:
//  * timestamp: RFC3339-formatted timestamp describing when the event happened.
//  * type: Describes the type of the operation which the event is related to. Values
//    can be apply, status, prune, delete, or error.
//  * eventType: Describes the type of the event. The set of possible values depends on the
//    the value of the type field.
//
// All the different types have a similar structure, with the exception of the
// error type. There is one event type pertaining to a specific resource, it being that
// it is applied, pruned or its status was updated, and there is a separate event type
// when all resources have been applied, deleted and so on. For any event that
// pertains to a particular resource, the fields group, kind, name and namespace
// will always be present.
//
// Events of type apply can have two different values for eventType, each which comes
// with a specific set of fields:
//  * resourceApplied: A resource has been applied to the cluster.
//    * fields identifying the resource.
//    * operation: The operation that was performed on the resource. Must be one of
//      created, configured, unchanged and serversideApplied.
//  * completed: All resources have been applied.
//    * count: Total number of resources applied
//    * createdCount: Number of resources created.
//    * configuredCount: Number of resources configured.
//    * unchangedCount: Number of resources unchanged.
//    * serversideAppliedCount: Number of resources applied serverside.
//
// Events of type status is a notification when either the status of resource
// has changed, or when a set of resources has reached their desired status. Events
// of type status can have three different values for eventType:
//  * resourceStatus: The status has changed for a resource.
//    * fields identifying the resource.
//    * status: The new status for the resource.
//    * message: Text that provides more information about the resource status.
//  * completed: All resources have reached the desired status.
//  * error: An error occurred when trying to get the status for a resource.
//    * fields identifying the resource.
//    * error: The error message.
//
// Events of type prune can have two different values for eventType, each which comes
// with a specific set of fields:
//  * resourcePruned: A resource has been pruned or was intended to be pruned but has been
//    skipped due to the presence of a lifecycle directive.
//    * fields identifying the resource.
//    * operation: The operation that was performed on the resource. Must be one
//      of pruned or skipped.
//  * completed: All resources have been pruned or skipped.
//    * count: Total number of resources pruned or skipped.
//    * prunedCount: Number of resources pruned.
//    * skippedCount: Number of resources skipped.
//
// Events of type delete can have two different values for eventType, each which comes
// with a specific set of fields:
//  * resourceDeleted: A resource has been deleted or was intended to be deleted but has been
//    skipped due to the presence of a lifecycle directive.
//    * fields identifying the resource.
//    * operation: The operation that was performed on the resource. Must be one
//      of deleted or skipped.
//  * completed: All resources have been deleted or skipped.
//    * count: Total number of resources deleted or skipped.
//    * deletedCount: Number of resources deleted.
//    * skippedCount: Number of resources skipped.
//
// Events of type error means there is an unrecoverable error and further
// processing will stop. Only a single value for eventType is possible:
//  * error: A fatal error has happened.
//    * error: The error message.
package json
