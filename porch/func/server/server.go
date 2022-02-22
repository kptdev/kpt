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
	"flag"
	"fmt"
	"net"
	"os"

	pb "github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"github.com/GoogleContainerTools/kpt/porch/func/internal"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

var (
	port      = flag.Int("port", 9445, "The server port")
	functions = flag.String("functions", "./functions", "Path to cached functions.")
	config    = flag.String("config", "./config.yaml", "Path to the config file.")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	address := fmt.Sprintf(":%d", *port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	evaluator, err := internal.NewEvaluatorWithConfig(*functions, *config)
	if err != nil {
		return fmt.Errorf("failed to initialize evaluator server: %w", err)
	}

	klog.Infof("Listening on %s", address)

	// Start the gRPC server
	server := grpc.NewServer()
	pb.RegisterFunctionEvaluatorServer(server, evaluator)
	if err := server.Serve(lis); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}
