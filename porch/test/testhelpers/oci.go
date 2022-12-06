package testhelpers

/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"context"
	"net"
	"net/http"
	"os"

	"github.com/GoogleContainerTools/kpt/porch/test/ociserver/pkg/oci"
)

func (h *Harness) StartOCIServer() *oci.Server {
	baseDir, err := os.MkdirTemp("", "repo-*")
	if err != nil {
		h.Fatalf("failed to create temporary directory for mock OCI server: %v", err)
	}

	var registryOptions []oci.RegistryOption
	registries := oci.NewDynamicRegistries(baseDir, registryOptions)

	server, err := oci.NewServer(registries)
	if err != nil {
		h.Fatalf("filed to initialize OCI server: %v", err)
	}

	ctx, cancel := context.WithCancel(h.Ctx)
	h.Cleanup(func() {
		cancel()
	})

	addressChannel := make(chan net.Addr)

	go func() {
		if err := server.ListenAndServe(ctx, "127.0.0.1:0", addressChannel); err != nil && err != http.ErrServerClosed {
			h.Errorf("Listen failed: %v", err)
		}
	}()

	address := <-addressChannel
	h.Logf("mock oci server listening on %s", address)

	return server
}
