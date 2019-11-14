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

// Package cmdbless contains the bless command
package cmdbless

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
	"lib.kpt.dev/kptfile"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// NewRunner returns a command runner.
func NewRunner() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "bless DIR",
		Short: "Initialize suggested package meta for a local config directory",
		Long: `Initialize suggested package meta for a local config directory.

Any directory containing Kubernetes Resource Configuration may be treated as
remote package without the existence of additional packaging metadata.

* Resource Configuration may be placed anywhere under DIR as *.yaml files.
* DIR may contain additional non-Resource Configuration files.
* DIR must be pushed to a git repo or repo subdirectory.

Bless will augment an existing local directory with packaging metadata to help
with discovery.

Bless will:

* Create a Kptfile with package name and metadata if it doesn't exist
* Create a Man.md for package documentation if it doesn't exist

Args:

  DIR:
    Defaults to '.'
    Bless fails if DIR does not exist`,
		Example: `
	# writes suggested package meta if not found
	kpt bless ./ --tag kpt.dev/app=cockroachdb --description "my cockroachdb implementation"`,
		RunE:         r.runE,
		SilenceUsage: true,
		PreRunE:      r.preRunE,
		Args:         cobra.ExactArgs(1),
	}

	c.Flags().StringVar(&r.Description, "description", "sample description", "short description of the package.")
	c.Flags().StringVar(&r.Name, "name", "", "package name.  defaults to the directory base name.")
	c.Flags().StringSliceVar(&r.Tags, "tag", []string{}, "list of tags for the package.")
	c.Flags().StringVar(&r.Url, "url", "", "link to page with information about the package.")
	r.Command = c
	return r
}

func NewCommand() *cobra.Command {
	return NewRunner().Command
}

// Runner contains the run function
type Runner struct {
	Command     *cobra.Command
	Tags        []string
	Name        string
	Description string
	Url         string
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"."}
	}
	if r.Name == "" {
		r.Name = filepath.Base(args[0])
	}
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	var err error
	if _, err = os.Stat(args[0]); os.IsNotExist(err) {
		return fmt.Errorf("%s does not exist", err)
	}

	if _, err = os.Stat(filepath.Join(args[0], "Kptfile")); os.IsNotExist(err) {
		fmt.Fprintf(c.OutOrStdout(), "writing %s\n", filepath.Join(args[0], "Kptfile"))
		k := kptfile.KptFile{
			ResourceMeta: yaml.ResourceMeta{ObjectMeta: yaml.ObjectMeta{Name: r.Name}},
			PackageMeta: kptfile.PackageMeta{
				ShortDescription: r.Description,
				Url:              r.Url,
				Tags:             r.Tags,
			},
		}

		// serialize the gvk when writing the Kptfile
		k.Kind = kptfile.TypeMeta.Kind
		k.ApiVersion = kptfile.TypeMeta.ApiVersion

		err = func() error {
			f, err := os.Create(filepath.Join(args[0], "Kptfile"))
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

	if _, err = os.Stat(filepath.Join(args[0], "MAN.md")); os.IsNotExist(err) {
		fmt.Fprintf(c.OutOrStdout(), "writing %s\n", filepath.Join(args[0], "MAN.md"))
		buff := &bytes.Buffer{}
		t, err := template.New("man").Parse(manTemplate)
		if err != nil {
			return err
		}

		err = t.Execute(buff, r)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(filepath.Join(args[0], "MAN.md"), buff.Bytes(), 0600)
		if err != nil {
			return err
		}
	}

	return nil
}

var manTemplate = `{{.Name}}
==================================================

# NAME

  {{.Name}}

# SYNOPSIS

  kubectl apply --recursive -f {{.Name}}

# Description

{{.Description}}

# SEE ALSO

`
