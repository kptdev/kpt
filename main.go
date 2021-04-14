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

//go:generate $GOBIN/mdtogo site/reference/live internal/docs/generated/livedocs --license=none --recursive=true --strategy=cmdDocs
//go:generate $GOBIN/mdtogo site/reference/pkg internal/docs/generated/pkgdocs --license=none --recursive=true --strategy=cmdDocs
//go:generate $GOBIN/mdtogo site/reference/fn internal/docs/generated/fndocs --license=none --recursive=true --strategy=cmdDocs
//go:generate $GOBIN/mdtogo site/reference internal/docs/generated/overview --license=none --strategy=cmdDocs
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/run"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog"
	"k8s.io/kubectl/pkg/util/logs"
	cliutilserror "sigs.k8s.io/cli-utils/pkg/errors"
)

func main() {
	var logFlags flag.FlagSet
	var err error

	ctx := context.Background()

	cmd := run.GetMain(ctx)
	logs.InitLogs()
	defer func() {
		logs.FlushLogs()
		if err != nil {
			os.Exit(1)
		}
	}()

	// Enable commandline flags for klog.
	// logging will help in collecting debugging information from users
	// Note(droot): There are too many flags exposed that makes the command
	// usage verbose but couldn't find a way to make it less verbose.
	klog.InitFlags(&logFlags)
	cmd.Flags().AddGoFlagSet(&logFlags)
	// By default klog v1 logs to stderr, switch that off
	_ = cmd.Flags().Set("logtostderr", "false")
	_ = cmd.Flags().Set("alsologtostderr", "false")

	err = cmd.Execute()
	if err != nil {
		handleErr(cmd, err)
	}
}

func handleErr(cmd *cobra.Command, err error) {
	var kptErr *errors.Error

	if errors.As(err, &kptErr) {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s \n", kptErr.Error())
		return
	}
	// fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", err)
	cmdutil.PrintErrorStacktrace(err)
	// TODO: find a way to avoid having to provide `kpt live` as a
	// parameter here.
	cliutilserror.CheckErr(cmd.ErrOrStderr(), err, "kpt live")
}
