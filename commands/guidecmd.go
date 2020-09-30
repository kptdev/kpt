// Copyright 2019 Google LLC
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

package commands

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/GoogleContainerTools/kpt/internal/guides/generated/consumer"
	"github.com/GoogleContainerTools/kpt/internal/guides/generated/ecosystem"
	"github.com/GoogleContainerTools/kpt/internal/guides/generated/producer"
	"github.com/spf13/cobra"
)

type guideInfo struct {
	Name        string
	Description string
	Content     string
}

var (
	guides = map[string][]guideInfo{
		"consumer": {
			{
				Name:        "Get",
				Description: "Get a remote package",
				Content:     consumer.GetGuide,
			},
			{
				Name:        "Update",
				Description: "Update a local package",
				Content:     consumer.UpdateGuide,
			},
			{
				Name:        "Set",
				Description: "Set field values",
				Content:     consumer.SetGuide,
			},
			{
				Name:        "Substitute",
				Description: "Substitute values into fields",
				Content:     consumer.SubstituteGuide,
			},
			{
				Name:        "Display",
				Description: "Display local package contents",
				Content:     consumer.DisplayGuide,
			},
			{
				Name:        "Apply",
				Description: "Apply a local package",
				Content:     consumer.ApplyGuide,
			},
			{
				Name:        "Function",
				Description: "Running functions",
				Content:     consumer.FunctionGuide,
			},
		},
		"producer": {
			{
				Name:        "Init",
				Description: "Init",
				Content:     producer.InitGuide,
			},
			{
				Name:        "Setters",
				Description: "Create setters",
				Content:     producer.SettersGuide,
			},
			{
				Name:        "Substitutions",
				Description: "Create substitutions",
				Content:     producer.SubstitutionsGuide,
			},
			{
				Name:        "Packages",
				Description: "Publishing a package",
				Content:     producer.PackageGuide,
			},
			{
				Name:        "Variants",
				Description: "Publishing variants",
				Content:     producer.VariantGuide,
			},
			{
				Name:        "Bootstrapping",
				Description: "Bootstrapping",
				Content:     producer.BootstrapGuide,
			},
		},
		"ecosystem": {
			{
				Name:        "Kustomize",
				Description: "Kustomize",
				Content:     ecosystem.KustomizeGuide,
			},
			{
				Name:        "Helm",
				Description: "Helm",
				Content:     ecosystem.HelmGuide,
			},
			{
				Name:        "OAM",
				Description: "OAM",
				Content:     ecosystem.OamGuide,
			},
		},
	}
)

func GetGuideCommand(name string) *cobra.Command {
	guide := &cobra.Command{
		Use:   "guide [NAME]",
		Short: `Print kpt guides`,
		Long:  getLongDescription(),
		Args:  cobra.ExactArgs(1),
		Example: `
  # Print the Apply tutorial
  kpt guide Apply`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			guide, found := findGuide(name)
			if !found {
				fmt.Fprintf(cmd.ErrOrStderr(), "unknown guide %q\n", name)
				os.Exit(1)
			}
			fmt.Fprintf(cmd.OutOrStdout(), guide.Content)
			return nil
		},
	}
	return guide
}

func findGuide(name string) (guideInfo, bool) {
	for _, gs := range guides {
		for _, g := range gs {
			if g.Name == name {
				return g, true
			}
		}
	}
	return guideInfo{}, false
}

func getLongDescription() string {
	tmpl := template.Must(template.New("longdesc").Parse(`
The kpt guides provide walkthroughs of common kpt workflows.

The following guides are available:

  Package Consumers:
  {{ range .consumer -}}
  - {{ .Name }}: {{ .Description }}
  {{ end }}
  Package Publishers:
  {{ range .producer -}}
  - {{ .Name }}: {{ .Description }}
  {{ end }}
  Ecosystem:
  {{ range .ecosystem -}}
  - {{ .Name }}: {{ .Description }}
  {{ end }}

`))

	var b bytes.Buffer
	err := tmpl.Execute(&b, guides)
	if err != nil {
		panic(err)
	}
	return b.String()
}
