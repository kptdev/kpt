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

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/GoogleContainerTools/kpt/porch/pkg/git"
	"k8s.io/klog/v2"
)

var (
	port = flag.Int("port", 9446, "Server port")
)

func main() {
	klog.InitFlags(nil)

	flag.Parse()

	if err := run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v", err)
	}
}

func run(dirs []string) error {
	var baseDir string

	switch len(dirs) {
	case 0:
		var err error
		baseDir, err = os.MkdirTemp("", "repo-*")
		if err != nil {
			return fmt.Errorf("failed to create temporary directory for git repository: %w", err)
		}

	case 1:
		baseDir = dirs[0]

	default:
		return fmt.Errorf("can serve only one git repository, not %d", len(dirs))
	}

	var gitRepoOptions []git.GitRepoOption
	repos := git.NewDynamicRepos(baseDir, gitRepoOptions)

	server, err := git.NewGitServer(repos)
	if err != nil {
		return fmt.Errorf("filed to initialize git server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addressChannel := make(chan net.Addr)

	go func() {
		if err := server.ListenAndServe(ctx, fmt.Sprintf(":%d", *port), addressChannel); err != nil && err != http.ErrServerClosed {
			klog.Fatalf("Listen failed: %v", err)
		}
	}()

	address := <-addressChannel
	fmt.Fprintf(os.Stderr, "Listening on %s\n", address)

	wait := make(chan os.Signal, 1)
	signal.Notify(wait, os.Interrupt)

	<-wait

	return nil
}
