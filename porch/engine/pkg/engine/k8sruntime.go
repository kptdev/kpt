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

package engine

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewKubeFunctionRuntime(kubeClient client.Client, namespace string) (*kubeFunctionRuntime, error) {
	return &kubeFunctionRuntime{
		kubeClient: kubeClient,
		namespace:  namespace,
	}, nil
}

type kubeFunctionRuntime struct {
	// kubeClient is the kubernetes client
	kubeClient client.Client

	// namespace holds the namespace where the executors run
	namespace string
}

var _ kpt.FunctionRuntime = &kubeFunctionRuntime{}

func (k *kubeFunctionRuntime) GetRunner(ctx context.Context, fn *v1.Function) (fn.FunctionRunner, error) {
	image := fn.Image

	tokens := strings.SplitN(image, ":", 2)
	if len(tokens) != 2 {
		// TODO: Assume latest?
		return nil, fmt.Errorf("expected version in image %q", image)
	}

	functionName := lastComponent(tokens[0])

	// Using labels here is a WIP experiment.
	// In theory it allows a pod/image to support multiple functions ... but that raises a discovery/trust issues
	// It might be easier simply to look at the image that the pod is running
	// However, today this is handy because it lets us run functions from a mirror, i.e. we can
	// give them a stable name in the package, but push them to our per-project gcr.io mirror.
	// TODO: Review use of labels if/when we start auto-launching pods.

	matchLabels := map[string]string{
		"functions.porch.kpt.dev/" + functionName: "",
	}
	// Work around limits of annotation/label keys/values
	matchAnnotations := map[string]string{
		"functions.porch.kpt.dev/" + functionName: image,
	}

	var pods corev1.PodList

	var options []client.ListOption
	options = append(options, client.InNamespace(k.namespace))
	options = append(options, client.MatchingLabels(matchLabels))
	if err := k.kubeClient.List(ctx, &pods, options...); err != nil {
		return nil, fmt.Errorf("error listing pods: %w", err)
	}

	// TODO: we should launch/manage the pods, rather than trusting the labels
	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.Status.Phase != "Running" {
			continue
		}

		matchesAnnotations := true
		for k, v := range matchAnnotations {
			if pod.Annotations[k] != v {
				matchesAnnotations = false
				break
			}
		}
		if !matchesAnnotations {
			break
		}
		return &kubeFunctionRunner{
			pod:   pod,
			image: image,
		}, nil
	}

	return nil, fmt.Errorf("could not find pod to run function %q", image)
}

func lastComponent(s string) string {
	lastSlash := strings.LastIndex(s, "/")
	return s[lastSlash+1:]
}

func (k *kubeFunctionRuntime) Close() error {
	return nil
}

type kubeFunctionRunner struct {
	pod   *corev1.Pod
	image string
}

var _ fn.FunctionRunner = &kubeFunctionRunner{}

func (k *kubeFunctionRunner) Run(r io.Reader, w io.Writer) error {
	// We shouldn't be putting this into the runner
	ctx := context.TODO()

	address := k.pod.Status.PodIP
	if address == "" {
		return fmt.Errorf("pod did not have podIP")
	}
	address += ":8888"

	klog.Infof("dialing grpc function runner %q", address)

	// TODO: pool connections
	cc, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to dial grpc function evaluator on %q for pod %s/%s: %w", address, k.pod.Namespace, k.pod.Name, err)
	}
	defer func() {
		if err := cc.Close(); err != nil {
			klog.Warningf("failed to close grpc connection: %v", err)
		}
	}()

	client := evaluator.NewFunctionEvaluatorClient(cc)

	in, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read function runner input: %w", err)
	}

	res, err := client.EvaluateFunction(ctx, &evaluator.EvaluateFunctionRequest{
		ResourceList: in,
		Image:        k.image,
	})
	if err != nil {
		return fmt.Errorf("func eval failed: %w", err)
	}
	if _, err := w.Write(res.ResourceList); err != nil {
		return fmt.Errorf("failed to write function runner output: %w", err)
	}
	return nil
}
