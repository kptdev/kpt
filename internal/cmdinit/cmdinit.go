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

// Package cmdinit contains the init command
package cmdinit

import (
	"bytes"
	"context"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/man"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, parent string) *Runner {
	r := &Runner{
		Ctx: ctx,
	}
	c := &cobra.Command{
		Use:     "init [DIR]",
		Args:    cobra.MaximumNArgs(1),
		Short:   docs.InitShort,
		Long:    docs.InitShort + "\n" + docs.InitLong,
		Example: docs.InitExamples,
		RunE:    r.runE,
	}

	c.Flags().StringVar(&r.Description, "description", "sample description", "short description of the package.")
	c.Flags().StringSliceVar(&r.Keywords, "keywords", []string{}, "list of keywords for the package.")
	c.Flags().StringVar(&r.Site, "site", "", "link to page with information about the package.")
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

func NewCommand(ctx context.Context, parent string) *cobra.Command {
	return NewRunner(ctx, parent).Command
}

// Runner contains the run function
type Runner struct {
	Command     *cobra.Command
	Keywords    []string
	Name        string
	Description string
	Site        string
	Ctx         context.Context
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = append(args, pkg.CurDir)
	}
	p, err := pkg.New(args[0], pkg.NewPkgFS(args[0], os.DirFS(args[0])))
	if err != nil {
		return err
	}
	r.Name = string(p.DisplayPath)

	up := string(p.UniquePath)
	if _, err = os.Stat(string(p.UniquePath)); os.IsNotExist(err) {
		return errors.Errorf("%s does not exist", err)
	}

	pr := printer.FromContextOrDie(r.Ctx)

	if _, err = os.Stat(filepath.Join(up, kptfilev1.KptFileName)); os.IsNotExist(err) {
		pr.Printf("writing %s\n", filepath.Join(args[0], "Kptfile"))
		k := kptfilev1.KptFile{
			ResourceMeta: yaml.ResourceMeta{
				ObjectMeta: yaml.ObjectMeta{
					NameMeta: yaml.NameMeta{
						Name: r.Name,
					},
				},
			},
			Info: &kptfilev1.PackageInfo{
				Description: r.Description,
				Site:        r.Site,
				Keywords:    r.Keywords,
			},
		}

		// serialize the gvk when writing the Kptfile
		k.Kind = kptfilev1.TypeMeta.Kind
		k.APIVersion = kptfilev1.TypeMeta.APIVersion

		err = func() error {
			f, err := os.Create(filepath.Join(up, kptfilev1.KptFileName))
			if err != nil {
				return err
			}
			defer f.Close()
			e := yaml.NewEncoder(f)

			defer e.Close()
			return e.Encode(k)
		}()
		if err != nil {
			return err
		}
	}

	if _, err = os.Stat(filepath.Join(up, man.ManFilename)); os.IsNotExist(err) {
		pr.Printf("writing %s\n", filepath.Join(args[0], man.ManFilename))
		buff := &bytes.Buffer{}
		t, err := template.New("man").Parse(manTemplate)
		if err != nil {
			return err
		}

		err = t.Execute(buff, r)
		if err != nil {
			return err
		}

		// Replace single quotes with backticks.
		content := strings.ReplaceAll(buff.String(), "'", "`")

		err = ioutil.WriteFile(filepath.Join(up, man.ManFilename), []byte(content), 0600)
		if err != nil {
			return err
		}
	}

	return nil
}

// manTemplate is the content for the automatically generated README.md file.
// It uses ' instead of ` since golang doesn't allow using ` in a raw string
// literal. We do a replace on the content before printing.
var manTemplate = `# {{.Name}}

## Description
{{.Description}}

## Usage

### Fetch the package
'kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] {{.Name}}'
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
'kpt pkg tree {{.Name}}'
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
'''
kpt live init {{.Name}}
kpt live apply {{.Name}} --reconcile-timeout=2m --output=table
'''
Details: https://kpt.dev/reference/cli/live/
`
