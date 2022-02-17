// Copyright 2022 Google LLC
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

	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/git"
	gogit "github.com/go-git/go-git/v5"
	"k8s.io/klog/v2"
)

var (
	port = flag.Int("port", 9446, "Server port")
)

func main() {
	flag.Parse()

	if err := run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v", err)
	}
}

func run(dirs []string) error {
	if len(dirs) != 1 {
		return fmt.Errorf("Expected one path to Git directory to serve. Got %d", len(dirs))
	}

	dir := dirs[0]

	var repo *gogit.Repository
	var err error

	if repo, err = gogit.PlainOpen(dir); err != nil {
		if err != gogit.ErrRepositoryNotExists {
			return fmt.Errorf("failed to open git repository %q: %w", dir, err)
		}
		isBare := true
		repo, err = gogit.PlainInit(dir, isBare)
		if err != nil {
			return fmt.Errorf("failed to initialize git repository %q: %w", dir, err)
		}
	}

	server, err := git.NewGitServer(repo)
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

	wait := make(chan os.Signal)
	signal.Notify(wait, os.Interrupt, os.Kill)

	<-wait

	return nil
}
