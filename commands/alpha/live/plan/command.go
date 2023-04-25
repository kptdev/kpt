// Copyright 2022 The kpt Authors
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

package plan

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	kptplanner "github.com/GoogleContainerTools/kpt/pkg/live/planner"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/cmd/flagutils"
	"sigs.k8s.io/cli-utils/pkg/common"
	print "sigs.k8s.io/cli-utils/pkg/print/common"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	TextOutput = "text"
	KRMOutput  = "krm"

	EntryPrefix   = "\t"
	ContentPrefix = "\t\t"
)

func NewRunner(ctx context.Context, factory util.Factory, ioStreams genericclioptions.IOStreams) *Runner {
	r := &Runner{
		ctx:       ctx,
		factory:   factory,
		ioStreams: ioStreams,
		serverSideOptions: common.ServerSideOptions{
			ServerSideApply: true,
		},
	}
	c := &cobra.Command{
		Use:     "plan [PKG_PATH | -]",
		PreRunE: r.PreRunE,
		RunE:    r.RunE,
	}
	c.Flags().StringVar(&r.inventoryPolicyString, flagutils.InventoryPolicyFlag, flagutils.InventoryPolicyStrict,
		"It determines the behavior when the resources don't belong to current inventory. Available options "+
			fmt.Sprintf("%q and %q.", flagutils.InventoryPolicyStrict, flagutils.InventoryPolicyAdopt))
	c.Flags().BoolVar(&r.serverSideOptions.ForceConflicts, "force-conflicts", false,
		"If true, overwrite applied fields on server if field manager conflict.")
	c.Flags().StringVar(&r.serverSideOptions.FieldManager, "field-manager", common.DefaultFieldManager,
		"The client owner of the fields being applied on the server-side.")
	c.Flags().StringVar(&r.output, "output", "text",
		"The output format for the plan. Must be either 'text' or 'krm'. Default is 'text'")
	r.Command = c

	return r
}

func NewCommand(ctx context.Context, factory util.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	return NewRunner(ctx, factory, ioStreams).Command
}

type Runner struct {
	ctx       context.Context
	Command   *cobra.Command
	factory   util.Factory
	ioStreams genericclioptions.IOStreams

	inventoryPolicyString string
	serverSideOptions     common.ServerSideOptions
	output                string
}

func (r *Runner) PreRunE(_ *cobra.Command, _ []string) error {
	return r.validateOutputFormat()
}

func (r *Runner) validateOutputFormat() error {
	if !(r.output == "text" || r.output == "krm") {
		return fmt.Errorf("unknown output format %q. Must be either 'text' or 'krm'", r.output)
	}
	return nil
}

func (r *Runner) RunE(c *cobra.Command, args []string) error {
	// default to the current working directory if the user didn't
	// provide a target package.
	if len(args) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		args = append(args, cwd)
	}

	// Handle symlinks.
	path := args[0]
	var err error
	if args[0] != "-" {
		path, err = argutil.ResolveSymlink(r.ctx, path)
		if err != nil {
			return err
		}
	}

	// Load the resources from disk or stdin and extract the
	// inventory information.
	objs, inv, err := live.Load(r.factory, path, c.InOrStdin())
	if err != nil {
		return err
	}

	// Convert the inventory data input to the format required by
	// the actuation code.
	invInfo, err := live.ToInventoryInfo(inv)
	if err != nil {
		return err
	}

	// Create and execute the planner.
	planner, err := kptplanner.NewClusterPlanner(r.factory)
	if err != nil {
		return err
	}
	plan, err := planner.BuildPlan(r.ctx, invInfo, objs, kptplanner.Options{
		ServerSideOptions: r.serverSideOptions,
	})
	if err != nil {
		return err
	}

	switch r.output {
	case "text":
		return printText(plan, objs, r.ioStreams)
	case "krm":
		return printKRM(plan, r.ioStreams)
	}
	return fmt.Errorf("unknown output format %s", r.output)
}

func printText(plan *kptplanner.Plan, objs []*unstructured.Unstructured, ioStreams genericclioptions.IOStreams) error {
	if !hasChanges(plan) {
		fmt.Fprint(ioStreams.Out, "no changes found\n")
		return nil
	}

	fmt.Fprintf(ioStreams.Out, "kpt will perform the following actions:\n")
	for i := range plan.Actions {
		action := plan.Actions[i]
		switch action.Type {
		case kptplanner.Create:
			printEntryWithColor("+", print.GREEN, action, ioStreams)
			u, ok := findResource(objs, action.Group, action.Kind, action.Namespace, action.Name)
			if !ok {
				panic("can't find resource")
			}
			printKRMWithPrefix(u, ContentPrefix, ioStreams)
		case kptplanner.Unchanged:
			// Do nothing.
		case kptplanner.Delete:
			printEntryWithColor("-", print.RED, action, ioStreams)
		case kptplanner.Update:
			printEntry(" ", action, ioStreams)
			findAndPrintDiff(action.Original, action.Updated, ContentPrefix, ioStreams)
		case kptplanner.Skip:
			// TODO: provide more information about why the resource was skipped.
			printEntryWithColor("=", print.YELLOW, action, ioStreams)
		case kptplanner.Error:
			printEntry("!", action, ioStreams)
			printWithPrefix(action.Error, ContentPrefix, ioStreams)
		}
		fmt.Fprintf(ioStreams.Out, "\n")
	}
	return nil
}

func hasChanges(plan *kptplanner.Plan) bool {
	for _, a := range plan.Actions {
		if a.Type != kptplanner.Unchanged {
			return true
		}
	}
	return false
}

func printEntryWithColor(prefix string, color print.Color, action kptplanner.Action, ioStreams genericclioptions.IOStreams) {
	txt := print.SprintfWithColor(color, "%s%s %s/%s %s/%s\n", EntryPrefix, prefix, action.Group, action.Kind, action.Namespace, action.Name)
	fmt.Fprint(ioStreams.Out, txt)
}

func printEntry(prefix string, action kptplanner.Action, ioStreams genericclioptions.IOStreams) {
	fmt.Fprintf(ioStreams.Out, "%s%s %s/%s %s/%s\n", EntryPrefix, prefix, action.Group, action.Kind, action.Namespace, action.Name)
}

func findAndPrintDiff(before, after *unstructured.Unstructured, prefix string, ioStreams genericclioptions.IOStreams) {
	diff, err := diffObjects(before, after)
	if err != nil {
		panic(err)
	}
	for _, d := range diff {
		if d.Path == ".metadata.generation" || d.Path == ".metadata.managedFields.0.time" {
			continue
		}

		switch d.Type {
		case "LeftAdd":
			txt := print.SprintfWithColor(print.RED, "%s-%s: %s\n", prefix, d.Path, strings.TrimSpace(fmt.Sprintf("%v", d.Left)))
			fmt.Fprint(ioStreams.Out, txt)
		case "RightAdd":
			txt := print.SprintfWithColor(print.GREEN, "%s+%s: %s\n", prefix, d.Path, strings.TrimSpace(fmt.Sprintf("%v", d.Right)))
			fmt.Fprint(ioStreams.Out, txt)
		case "Change":
			txt1 := print.SprintfWithColor(print.RED, "%s-%s: %s\n", prefix, d.Path, strings.TrimSpace(fmt.Sprintf("%v", d.Left)))
			fmt.Fprint(ioStreams.Out, txt1)
			txt2 := print.SprintfWithColor(print.GREEN, "%s+%s: %s\n", prefix, d.Path, strings.TrimSpace(fmt.Sprintf("%v", d.Right)))
			fmt.Fprint(ioStreams.Out, txt2)
		}
	}
}

func findResource(objs []*unstructured.Unstructured, group, kind, namespace, name string) (*unstructured.Unstructured, bool) {
	for i := range objs {
		o := objs[i]
		gvk := o.GroupVersionKind()
		if gvk.Group == group && gvk.Kind == kind && o.GetName() == name && o.GetNamespace() == namespace {
			return o, true
		}
	}
	return nil, false
}

func printKRMWithPrefix(u *unstructured.Unstructured, prefix string, ioStreams genericclioptions.IOStreams) {
	b, err := yaml.Marshal(u.Object)
	if err != nil {
		panic(fmt.Errorf("unable to marshal resource: %v", err))
	}
	printWithPrefix(string(b), prefix, ioStreams)
}

func printWithPrefix(text, prefix string, ioStreams genericclioptions.IOStreams) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		fmt.Fprintf(ioStreams.Out, "%s%s\n", prefix, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		panic(fmt.Errorf("error reading text: %v", err))
	}
}

// printKRM outputs the plan inside a ResourceList so the output format
// follows the KRM function wire format.
func printKRM(
	plan *kptplanner.Plan,
	ioStreams genericclioptions.IOStreams,
) error {
	planResource, err := yaml.Parse(strings.TrimSpace(`
apiVersion: kpt.dev/v1alpha1
kind: Plan
metadata:
  name: plan
  annotations:
    config.kubernetes.io/local-config: true	
`))
	if err != nil {
		return fmt.Errorf("unable to create yaml document: %w", err)
	}

	sNode, err := planResource.Pipe(yaml.LookupCreate(yaml.SequenceNode, "spec", "actions"))
	if err != nil {
		return fmt.Errorf("unable to update yaml document: %w", err)
	}

	for i := range plan.Actions {
		action := plan.Actions[i]
		a := yaml.NewRNode(&yaml.Node{Kind: yaml.MappingNode})
		fields := map[string]*yaml.RNode{
			"action":     yaml.NewScalarRNode(string(action.Type)),
			"apiVersion": yaml.NewScalarRNode(action.Group),
			"kind":       yaml.NewScalarRNode(action.Kind),
			"name":       yaml.NewScalarRNode(action.Name),
			"namespace":  yaml.NewScalarRNode(action.Namespace),
		}
		if action.Original != nil {
			r, err := unstructuredToRNode(action.Original)
			if err != nil {
				return err
			}
			fields["original"] = r
		}
		if action.Updated != nil {
			r, err := unstructuredToRNode(action.Updated)
			if err != nil {
				return err
			}
			fields["updated"] = r
		}
		if action.Error != "" {
			fields["error"] = yaml.NewScalarRNode(action.Error)
		}

		for key, val := range fields {
			if err := a.PipeE(yaml.SetField(key, val)); err != nil {
				return fmt.Errorf("unable to update yaml document: %w", err)
			}
		}
		if err := sNode.PipeE(yaml.Append(a.YNode())); err != nil {
			return fmt.Errorf("unable to update yaml document: %w", err)
		}
	}

	writer := &kio.ByteWriter{
		Writer:                ioStreams.Out,
		KeepReaderAnnotations: true,
		WrappingAPIVersion:    kio.ResourceListAPIVersion,
		WrappingKind:          kio.ResourceListKind,
	}
	err = writer.Write([]*yaml.RNode{planResource})
	if err != nil {
		return fmt.Errorf("failed to write resources: %w", err)
	}
	return nil
}

func unstructuredToRNode(u *unstructured.Unstructured) (*yaml.RNode, error) {
	b, err := yaml.Marshal(u.Object)
	if err != nil {
		return nil, err
	}
	return yaml.Parse((string(b)))
}
