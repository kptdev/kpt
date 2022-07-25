// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"context"
	"fmt"
	"os"
	ss "strings"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/strings"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/GoogleContainerTools/kpt/pkg/status"
	statusprinters "github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/status/printers"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/status/printers/list"
	"github.com/go-errors/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/slice"
	"sigs.k8s.io/cli-utils/pkg/apply/poller"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/aggregator"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/print/common"
	"sigs.k8s.io/cli-utils/pkg/printers"
)

const (
	Known   = "known"
	Current = "current"
	Deleted = "deleted"
	Forever = "forever"
)

const (
	Local  = "local"
	Remote = "remote"
)

var (
	PollUntilOptions = []string{Known, Current, Deleted, Forever}
)

func NewRunner(ctx context.Context, factory util.Factory) *Runner {
	r := &Runner{
		ctx:               ctx,
		pollerFactoryFunc: pollerFactoryFunc,
		invClientFunc:     invClient,
		factory:           factory,
	}
	c := &cobra.Command{
		Use:     "status [PKG_PATH | -]",
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Short:   livedocs.StatusShort,
		Long:    livedocs.StatusShort + "\n" + livedocs.StatusLong,
		Example: livedocs.StatusExamples,
	}
	r.Command = c
	c.Flags().DurationVar(&r.period, "poll-period", 2*time.Second,
		"Polling period for resource statuses.")
	c.Flags().StringVar(&r.pollUntil, "poll-until", "known",
		fmt.Sprintf("When to stop polling. Must be one of %s", strings.JoinStringsWithQuotes(PollUntilOptions)))
	c.Flags().StringVar(&r.output, "output", "events", "Output format.")
	c.Flags().DurationVar(&r.timeout, "timeout", 0,
		"How long to wait before exiting")
	c.Flags().StringVar(&r.invType, "inv-type", Local, "Type of the inventory info, must be local or remote")
	c.Flags().StringVar(&r.inventoryNames, "inv-names", "", "Names of targeted inventory: inv1,inv2,...")
	c.Flags().StringVar(&r.namespaces, "namespaces", "", "Names of targeted namespaces: ns1,ns2,...")
	c.Flags().StringVar(&r.statuses, "statuses", "", "Targeted status: st1,st2...")
	return r
}

func NewCommand(ctx context.Context, factory util.Factory) *cobra.Command {
	return NewRunner(ctx, factory).Command
}

// Runner captures the parameters for the command and contains
// the run function.
type Runner struct {
	ctx           context.Context
	Command       *cobra.Command
	factory       util.Factory
	invClientFunc func(util.Factory) (inventory.Client, error)

	period    time.Duration
	pollUntil string
	timeout   time.Duration
	output    string

	invType          string
	inventoryNames   string
	inventoryNameSet map[string]bool
	namespaces       string
	namespaceSet     map[string]bool
	statuses         string
	statusSet        map[string]bool

	pollerFactoryFunc func(util.Factory) (poller.Poller, error)
}

func (r *Runner) preRunE(*cobra.Command, []string) error {
	if !slice.ContainsString(PollUntilOptions, r.pollUntil, nil) {
		return fmt.Errorf("pollUntil must be one of %s",
			strings.JoinStringsWithQuotes(PollUntilOptions))
	}

	if found := printers.ValidatePrinterType(r.output); !found {
		return fmt.Errorf("unknown output type %q", r.output)
	}

	if r.invType != Local && r.invType != Remote {
		return fmt.Errorf("inv-type flag should be either local or remote")
	}

	if r.invType == Local && r.inventoryNames != "" {
		return fmt.Errorf("inv-names flag should only be used when inv-type is set to remote")
	}

	if r.inventoryNames != "" {
		r.inventoryNameSet = make(map[string]bool)
		for _, name := range ss.Split(r.inventoryNames, ",") {
			r.inventoryNameSet[name] = true
		}
	}

	if r.namespaces != "" {
		r.namespaceSet = make(map[string]bool)
		for _, ns := range ss.Split(r.namespaces, ",") {
			r.namespaceSet[ns] = true
		}
	}

	if r.statuses != "" {
		r.statusSet = make(map[string]bool)
		for _, st := range ss.Split(r.statuses, ",") {
			parsedST := ss.ToLower(st)
			r.statusSet[parsedST] = true
		}
	}
	return nil
}

// Load inventory info from local storage
// and get info from the cluster based on the local info
// wrap it to be a map mapping from string to objectMetadataSet
func (r *Runner) loadInvFromDisk(c *cobra.Command, args []string) (*list.PrintData, error) {
	if len(args) == 0 {
		// default to the current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		args = append(args, cwd)
	}

	path := args[0]
	var err error
	if args[0] != "-" {
		path, err = argutil.ResolveSymlink(r.ctx, path)
		if err != nil {
			return nil, err
		}
	}

	_, inv, err := live.Load(r.factory, path, c.InOrStdin())
	if err != nil {
		return nil, err
	}

	invInfo, err := live.ToInventoryInfo(inv)
	if err != nil {
		return nil, err
	}

	invClient, err := r.invClientFunc(r.factory)
	if err != nil {
		return nil, err
	}

	// Based on the inventory template manifest we look up the inventory
	// from the live state using the inventory client.
	identifiers, err := invClient.GetClusterObjs(invInfo)
	if err != nil {
		return nil, err
	}

	printData := list.PrintData{}
	// initialize maps in printData
	printData.InvNameMap = make(map[object.ObjMetadata]string)
	for _, obj := range identifiers {
		// check if the object is under one of the targeted namespaces
		if _, ok := r.namespaceSet[obj.Namespace]; ok || len(r.namespaceSet) == 0 {
			// add to the map for future reference
			printData.InvNameMap[obj] = inv.Name
			// append to identifiers
			printData.Identifiers = append(printData.Identifiers, obj)
		}
	}
	return &printData, nil
}

// Retrieve a list of inventory object from the cluster
// Wrap it to become a map mapping from string to ObjMetadataSet
// Refer to the backbone of GetClusterObjs function in inventory package
func (r *Runner) listInvFromCluster() (*list.PrintData, error) {
	// Create an emtpy PrintData object
	printData := list.PrintData{}
	// Launch a dynamic client
	dc, err := r.factory.DynamicClient()
	if err != nil {
		return nil, err
	}

	// Launch a mapper
	mapper, err := r.factory.ToRESTMapper()
	if err != nil {
		return nil, err
	}

	// Define the mapping
	mapping, err := mapper.RESTMapping(live.ResourceGroupGVK.GroupKind(), live.ResourceGroupGVK.Version)
	if err != nil {
		return nil, err
	}

	// retrieve the list from the cluster
	clusterInvs, err := dc.Resource(mapping.Resource).List(r.ctx, metav1.ListOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	if apierrors.IsNotFound(err) {
		return &printData, nil
	}

	// wrapping
	// initialize maps in printData
	printData.InvNameMap = make(map[object.ObjMetadata]string)

	for _, inv := range clusterInvs.Items {
		invName := inv.GetName()
		// Check if there are targeted inventory names and include the current inventory name
		if _, ok := r.inventoryNameSet[invName]; !ok && len(r.inventoryNameSet) != 0 {
			continue
		}

		// Get wrapped object
		wrappedInvObj := live.WrapInventoryObj(&inv)
		wrappedInvObjSlice, err := wrappedInvObj.Load()
		if err != nil {
			return nil, err
		}

		// Filter objects
		for _, obj := range wrappedInvObjSlice {
			// check if the object is under one of the targeted namespaces
			if _, ok := r.namespaceSet[obj.Namespace]; ok || len(r.namespaceSet) == 0 {
				// add to the map for future reference
				printData.InvNameMap[obj] = invName
				// append to identifiers
				printData.Identifiers = append(printData.Identifiers, obj)
			}
		}
	}
	return &printData, nil
}

// runE implements the logic of the command and will delegate to the
// poller to compute status for each of the resources. One of the printer
// implementations takes care of printing the output.
func (r *Runner) runE(c *cobra.Command, args []string) error {
	pr := printer.FromContextOrDie(r.ctx)

	var printData *list.PrintData
	var err error
	if r.invType == Remote {
		if len(args) != 0 {
			fmt.Printf("%c[%dm", common.ESC, common.YELLOW)
			fmt.Println("Warning: Path is assigned while list flag is enabled, ignore the path")
			fmt.Printf("%c[%dm", common.ESC, common.RESET)
		}
		printData, err = r.listInvFromCluster()
	} else {
		printData, err = r.loadInvFromDisk(c, args)
	}
	if err != nil {
		return err
	}

	// Exit here if the inventory is empty.
	if len(printData.Identifiers) == 0 {
		pr.Printf("no resources found in the inventory\n")
		return nil
	}

	// If status flag is assigned, include it to printData
	if len(r.statusSet) != 0 {
		printData.StatusSet = r.statusSet
	}

	statusPoller, err := r.pollerFactoryFunc(r.factory)
	if err != nil {
		return err
	}
	// If the user has specified a timeout, we create a context with timeout,
	// otherwise we create a context with cancel.
	var ctx context.Context
	var cancel func()
	if r.timeout != 0 {
		ctx, cancel = context.WithTimeout(r.ctx, r.timeout)
	} else {
		ctx, cancel = context.WithCancel(r.ctx)
	}
	defer cancel()

	// Choose the appropriate ObserverFunc based on the criteria for when
	// the command should exit.
	var cancelFunc collector.ObserverFunc
	switch r.pollUntil {
	case Known:
		cancelFunc = allKnownNotifierFunc(cancel)
	case Current:
		cancelFunc = desiredStatusNotifierFunc(cancel, kstatus.CurrentStatus)
	case Deleted:
		cancelFunc = desiredStatusNotifierFunc(cancel, kstatus.NotFoundStatus)
	case Forever:
		cancelFunc = func(*collector.ResourceStatusCollector, event.Event) {}
	default:
		return fmt.Errorf("unknown value for pollUntil: %q", r.pollUntil)
	}
	// Fetch a printer implementation based on the desired output format as
	// specified in the output flag.

	statusPrinter, err := statusprinters.CreatePrinter(r.output, genericclioptions.IOStreams{
		Out:    pr.OutStream(),
		ErrOut: pr.ErrStream(),
	}, printData)

	if err != nil {
		return errors.WrapPrefix(err, "error creating printer", 1)
	}

	eventChannel := statusPoller.Poll(ctx, printData.Identifiers, polling.PollOptions{
		PollInterval: r.period,
	})

	return statusPrinter.Print(eventChannel, printData.Identifiers, cancelFunc)
}

// desiredStatusNotifierFunc returns an Observer function for the
// ResourceStatusCollector that will cancel the context (using the cancelFunc)
// when all resources have reached the desired status.
func desiredStatusNotifierFunc(cancelFunc context.CancelFunc,
		desired kstatus.Status) collector.ObserverFunc {
	return func(rsc *collector.ResourceStatusCollector, _ event.Event) {
		var rss []*event.ResourceStatus
		for _, rs := range rsc.ResourceStatuses {
			rss = append(rss, rs)
		}
		aggStatus := aggregator.AggregateStatus(rss, desired)
		if aggStatus == desired {
			cancelFunc()
		}
	}
}

// allKnownNotifierFunc returns an Observer function for the
// ResourceStatusCollector that will cancel the context (using the cancelFunc)
// when all resources have a known status.
func allKnownNotifierFunc(cancelFunc context.CancelFunc) collector.ObserverFunc {
	return func(rsc *collector.ResourceStatusCollector, _ event.Event) {
		for _, rs := range rsc.ResourceStatuses {
			if rs.Status == kstatus.UnknownStatus {
				return
			}
		}
		cancelFunc()
	}
}

func pollerFactoryFunc(f util.Factory) (poller.Poller, error) {
	return status.NewStatusPoller(f)
}

func invClient(f util.Factory) (inventory.Client, error) {
	return inventory.NewClient(f, live.WrapInventoryObj, live.InvToUnstructuredFunc, inventory.StatusPolicyAll)
}
