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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/kptfile"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/man"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// NewRunner returns a command runner.
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:     "init DIR",
		Args:    cobra.ExactArgs(1),
		Short:   docs.InitShort,
		Long:    docs.InitShort + "\n" + docs.InitLong,
		Example: docs.InitExamples,
		RunE:    r.runE,
		PreRunE: r.preRunE,
	}

	c.Flags().StringVar(&r.Description, "description", "sample description", "short description of the package.")
	c.Flags().StringVar(&r.Name, "name", "", "package name.  defaults to the directory base name.")
	c.Flags().StringSliceVar(&r.Tags, "tag", []string{}, "list of tags for the package.")
	c.Flags().StringVar(&r.URL, "url", "", "link to page with information about the package.")
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).Command
}

// Runner contains the run function
type Runner struct {
	Command     *cobra.Command
	Tags        []string
	Name        string
	Description string
	URL         string
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
		return errors.Errorf("%s does not exist", err)
	}

	if _, err = os.Stat(filepath.Join(args[0], "Kptfile")); os.IsNotExist(err) {
		fmt.Fprintf(c.OutOrStdout(), "writing %s\n", filepath.Join(args[0], "Kptfile"))
		k := kptfile.KptFile{
			ResourceMeta: yaml.ResourceMeta{ObjectMeta: yaml.ObjectMeta{Name: r.Name}},
			PackageMeta: kptfile.PackageMeta{
				ShortDescription: r.Description,
				URL:              r.URL,
				Tags:             r.Tags,
			},
		}

		// serialize the gvk when writing the Kptfile
		k.Kind = kptfile.TypeMeta.Kind
		k.APIVersion = kptfile.TypeMeta.APIVersion

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

	if _, err = os.Stat(filepath.Join(args[0], man.ManFilename)); os.IsNotExist(err) {
		fmt.Fprintf(c.OutOrStdout(), "writing %s\n", filepath.Join(args[0], man.ManFilename))
		buff := &bytes.Buffer{}
		t, err := template.New("man").Parse(manTemplate)
		if err != nil {
			return err
		}

		err = t.Execute(buff, r)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(filepath.Join(args[0], man.ManFilename), buff.Bytes(), 0600)
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
