// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/internal/util/strings"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/status/printers"
	"github.com/go-errors/errors"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/slice"
	"sigs.k8s.io/cli-utils/pkg/apply/poller"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/aggregator"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/provider"
	"sigs.k8s.io/cli-utils/pkg/util/factory"
)

const (
	Known   = "known"
	Current = "current"
	Deleted = "deleted"
	Forever = "forever"
)

var (
	PollUntilOptions = []string{Known, Current, Deleted, Forever}
)

func NewRunner(ctx context.Context, provider provider.Provider,
	ioStreams genericclioptions.IOStreams) *Runner {
	r := &Runner{
		ctx:               ctx,
		provider:          provider,
		pollerFactoryFunc: pollerFactoryFunc,
		ioStreams:         ioStreams,
	}
	c := &cobra.Command{
		Use:     "status [PKG_PATH]",
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
	return r
}

func NewCommand(ctx context.Context, provider provider.Provider,
	ioStreams genericclioptions.IOStreams) *cobra.Command {
	return NewRunner(ctx, provider, ioStreams).Command
}

// Runner captures the parameters for the command and contains
// the run function.
type Runner struct {
	ctx       context.Context
	Command   *cobra.Command
	ioStreams genericclioptions.IOStreams
	provider  provider.Provider

	period    time.Duration
	pollUntil string
	timeout   time.Duration
	output    string

	pollerFactoryFunc func(util.Factory) (poller.Poller, error)
}

func (r *Runner) preRunE(*cobra.Command, []string) error {
	if !slice.ContainsString(PollUntilOptions, r.pollUntil, nil) {
		return fmt.Errorf("pollUntil must be one of %s",
			strings.JoinStringsWithQuotes(PollUntilOptions))
	}
	return nil
}

// runE implements the logic of the command and will delegate to the
// poller to compute status for each of the resources. One of the printer
// implementations takes care of printing the output.
func (r *Runner) runE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		// default to the current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		args = append(args, cwd)
	}

	_, err := common.DemandOneDirectory(args)
	if err != nil {
		return err
	}

	_, inv, err := live.Load(r.provider.Factory(), args[0], c.InOrStdin())
	if err != nil {
		return err
	}

	invInfo, err := live.ToInventoryInfo(inv)
	if err != nil {
		return err
	}

	invClient, err := r.provider.InventoryClient()
	if err != nil {
		return err
	}

	// Based on the inventory template manifest we look up the inventory
	// from the live state using the inventory client.
	identifiers, err := invClient.GetClusterObjs(invInfo)
	if err != nil {
		return err
	}

	// Exit here if the inventory is empty.
	if len(identifiers) == 0 {
		_, _ = fmt.Fprint(r.ioStreams.Out, "no resources found in the inventory\n")
		return nil
	}

	statusPoller, err := r.pollerFactoryFunc(r.provider.Factory())
	if err != nil {
		return err
	}

	// Fetch a printer implementation based on the desired output format as
	// specified in the output flag.
	printer, err := printers.CreatePrinter(r.output, r.ioStreams)
	if err != nil {
		return errors.WrapPrefix(err, "error creating printer", 1)
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
		cancelFunc = desiredStatusNotifierFunc(cancel, status.CurrentStatus)
	case Deleted:
		cancelFunc = desiredStatusNotifierFunc(cancel, status.NotFoundStatus)
	case Forever:
		cancelFunc = func(*collector.ResourceStatusCollector, event.Event) {}
	default:
		return fmt.Errorf("unknown value for pollUntil: %q", r.pollUntil)
	}

	eventChannel := statusPoller.Poll(ctx, identifiers, polling.Options{
		PollInterval: r.period,
		UseCache:     true,
	})

	printer.Print(eventChannel, identifiers, cancelFunc)
	return nil
}

// desiredStatusNotifierFunc returns an Observer function for the
// ResourceStatusCollector that will cancel the context (using the cancelFunc)
// when all resources have reached the desired status.
func desiredStatusNotifierFunc(cancelFunc context.CancelFunc,
	desired status.Status) collector.ObserverFunc {
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
			if rs.Status == status.UnknownStatus {
				return
			}
		}
		cancelFunc()
	}
}

func pollerFactoryFunc(f util.Factory) (poller.Poller, error) {
	return factory.NewStatusPoller(f)
}
