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
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	pb "github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	addressFlag = flag.String("address", "localhost:9445", "FunctionEvaluator server address")
	packageFlag = flag.String("package", "", "Source package")
	imageFlag   = flag.String("image", "", "Image of the function to evaluate")
)

func main() {
	flag.Parse()

	if err := run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	rl, err := createResourceList(args)
	if err != nil {
		return err
	}

	fmt.Printf("Request:\n\n%s\n", string(rl))

	res, err := call(rl)
	if err != nil {
		return err
	}

	fmt.Printf("Response:\n\n%s\n\nLog:\n%s\n", string(res.ResourceList), string(res.Log))
	return nil
}

func createResourceList(args []string) ([]byte, error) {
	r := kio.LocalPackageReader{
		PackagePath:        *packageFlag,
		IncludeSubpackages: true,
		WrapBareSeqNode:    true,
	}

	cfg, err := configmap(args)
	if err != nil {
		return nil, fmt.Errorf("failed to create function config: %w", err)
	}

	var b bytes.Buffer
	w := kio.ByteWriter{
		Writer:                &b,
		KeepReaderAnnotations: true,
		Style:                 0,
		FunctionConfig:        cfg,
		WrappingKind:          kio.ResourceListKind,
		WrappingAPIVersion:    kio.ResourceListAPIVersion,
	}

	if err := (kio.Pipeline{Inputs: []kio.Reader{r}, Outputs: []kio.Writer{w}}).Execute(); err != nil {
		return nil, fmt.Errorf("failed to create serialized ResourceList: %w", err)
	}

	return b.Bytes(), nil
}

func call(rl []byte) (*pb.EvaluateFunctionResponse, error) {
	cc, err := grpc.Dial(*addressFlag, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", *addressFlag, err)
	}
	defer cc.Close()

	evaluator := pb.NewFunctionEvaluatorClient(cc)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	in := &pb.EvaluateFunctionRequest{
		ResourceList: rl,
		Image:        *imageFlag,
	}

	r, err := evaluator.EvaluateFunction(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("function evaluation failed: %w", err)
	} else {
		return r, nil
	}
}

func configmap(args []string) (*yaml.RNode, error) {
	if len(args) == 0 {
		return nil, nil
	}

	data := map[string]string{}
	for _, a := range args {
		split := strings.SplitN(a, "=", 2)
		if len(split) != 2 {
			return nil, fmt.Errorf("invalid config value: %q", a)
		}
		data[split[0]] = split[1]
	}
	node := yaml.NewMapRNode(&data)
	if node == nil {
		return nil, nil
	}

	// create ConfigMap resource to contain function config
	configMap := yaml.MustParse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: function-input
data: {}
`)
	if err := configMap.PipeE(yaml.SetField("data", node)); err != nil {
		return nil, err
	}
	return configMap, nil
}
