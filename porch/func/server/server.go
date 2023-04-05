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
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	pb "github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"github.com/GoogleContainerTools/kpt/porch/func/healthchecker"
	"github.com/GoogleContainerTools/kpt/porch/func/internal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"k8s.io/klog/v2"
)

const (
	execRuntime = "exec"
	podRuntime  = "pod"

	wrapperServerImageEnv = "WRAPPER_SERVER_IMAGE"
)

var (
	port            = flag.Int("port", 9445, "The server port")
	functions       = flag.String("functions", "./functions", "Path to cached functions.")
	config          = flag.String("config", "./config.yaml", "Path to the config file.")
	podCacheConfig  = flag.String("pod-cache-config", "/pod-cache-config/pod-cache-config.yaml", "Path to the pod cache config file. The file is map of function name to TTL.")
	podNamespace    = flag.String("pod-namespace", "porch-fn-system", "Namespace to run KRM functions pods.")
	podTTL          = flag.Duration("pod-ttl", 30*time.Minute, "TTL for pods before GC.")
	scanInterval    = flag.Duration("scan-interval", time.Minute, "The interval of GC between scans.")
	disableRuntimes = flag.String("disable-runtimes", "", fmt.Sprintf("The runtime(s) to disable. Multiple runtimes should separated by `,`. Available runtimes: `%v`, `%v`.", execRuntime, podRuntime))
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

	availableRuntimes := map[string]struct{}{
		execRuntime: {},
		podRuntime:  {},
	}
	if disableRuntimes != nil {
		runtimesFromFlag := strings.Split(*disableRuntimes, ",")
		for _, rt := range runtimesFromFlag {
			delete(availableRuntimes, rt)
		}
	}
	runtimes := []internal.Evaluator{}
	for rt := range availableRuntimes {
		switch rt {
		case execRuntime:
			execEval, err := internal.NewExecutableEvaluator(*functions, *config)
			if err != nil {
				return fmt.Errorf("failed to initialize executable evaluator: %w", err)
			}
			runtimes = append(runtimes, execEval)
		case podRuntime:
			wrapperServerImage := os.Getenv(wrapperServerImageEnv)
			if wrapperServerImage == "" {
				return fmt.Errorf("environment variable %v must be set to use pod function evaluator runtime", wrapperServerImageEnv)
			}
			podEval, err := internal.NewPodEvaluator(*podNamespace, wrapperServerImage, *scanInterval, *podTTL, *podCacheConfig)
			if err != nil {
				return fmt.Errorf("failed to initialize pod evaluator: %w", err)
			}
			runtimes = append(runtimes, podEval)
		}
	}
	if len(runtimes) == 0 {
		klog.Warning("no runtime is enabled in function-runner")
	}
	evaluator := internal.NewMultiEvaluator(runtimes...)

	klog.Infof("Listening on %s", address)

	// Start the gRPC server
	server := grpc.NewServer()
	pb.RegisterFunctionEvaluatorServer(server, evaluator)
	healthService := healthchecker.NewHealthChecker()
	grpc_health_v1.RegisterHealthServer(server, healthService)
	if err := server.Serve(lis); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}
