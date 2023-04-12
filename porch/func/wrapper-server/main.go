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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	pb "github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"github.com/GoogleContainerTools/kpt/porch/func/healthchecker"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

func main() {
	op := &options{}
	cmd := &cobra.Command{
		Use:   "wrapper-server",
		Short: "wrapper-server is a gRPC server that fronts a KRM function",
		RunE: func(cmd *cobra.Command, args []string) error {
			argsLenAtDash := cmd.ArgsLenAtDash()
			if argsLenAtDash > -1 {
				op.entrypoint = args[argsLenAtDash:]
			}
			return op.run()
		},
	}
	cmd.Flags().IntVar(&op.port, "port", 9446, "The server port")
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
		os.Exit(1)
	}
}

type options struct {
	port       int
	entrypoint []string
}

func (o *options) run() error {
	address := fmt.Sprintf(":%d", o.port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	evaluator := &singleFunctionEvaluator{
		entrypoint: o.entrypoint,
	}

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

type singleFunctionEvaluator struct {
	pb.UnimplementedFunctionEvaluatorServer

	entrypoint []string
}

func (e *singleFunctionEvaluator) EvaluateFunction(ctx context.Context, req *pb.EvaluateFunctionRequest) (*pb.EvaluateFunctionResponse, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, e.entrypoint[0], e.entrypoint[1:]...)
	cmd.Stdin = bytes.NewReader(req.ResourceList)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	var exitErr *exec.ExitError
	outbytes := stdout.Bytes()
	stderrStr := stderr.String()
	if err != nil {
		if errors.As(err, &exitErr) {
			// If the exit code is non-zero, we will try to embed the structured results and content from stderr into the error message.
			rl, pe := fn.ParseResourceList(outbytes)
			if pe != nil {
				// If we can't parse the output resource list, we only surface the content in stderr.
				return nil, status.Errorf(codes.Internal, "failed to evaluate function %q with stderr %v", req.Image, stderrStr)
			}
			return nil, status.Errorf(codes.Internal, "failed to evaluate function %q with structured results: %v and stderr: %v", req.Image, rl.Results.Error(), stderrStr)
		} else {
			return nil, status.Errorf(codes.Internal, "Failed to execute function %q: %s (%s)", req.Image, err, stderrStr)
		}
	}

	klog.Infof("Evaluated %q: stdout length: %d\nstderr:\n%v", req.Image, len(outbytes), stderrStr)

	return &pb.EvaluateFunctionResponse{
		ResourceList: outbytes,
		Log:          []byte(stderrStr),
	}, nil
}
