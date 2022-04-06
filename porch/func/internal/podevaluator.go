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

package internal

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	pb "github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	defaultWrapperServerPort = 9446
	volumeName               = "wrapper-server-tools"
	volumeMountPath          = "/wrapper-server-tools"
	wrapperServerBin         = "wrapper-server"
	gRPCProbeBin             = "grpc-health-probe"
)

type podEvaluator struct {
	// kubeClient is the kubernetes client
	kubeClient client.Client
	// namespace holds the namespace where the executors run
	namespace string
	// wrapperServerImage is the image name of the wrapper server
	wrapperServerImage string
}

var _ Evaluator = &podEvaluator{}

func NewPodEvaluator(namespace, wrapperServerImage string) (Evaluator, error) {
	restCfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}
	cl, err := client.New(restCfg, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return &podEvaluator{
		kubeClient:         cl,
		namespace:          namespace,
		wrapperServerImage: wrapperServerImage,
	}, nil
}

func (pe *podEvaluator) EvaluateFunction(ctx context.Context, req *pb.EvaluateFunctionRequest) (*pb.EvaluateFunctionResponse, error) {
	ref, err := name.ParseReference(req.Image)
	if err != nil {
		return nil, err
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, err
	}
	// TODO: cache the config file
	cf, err := img.ConfigFile()
	cfg := cf.Config
	var fnCmd []string
	if len(cfg.Entrypoint) != 0 {
		fnCmd = cfg.Entrypoint
	} else {
		fnCmd = cfg.Cmd
	}

	cmd := append([]string{
		path.Join(volumeMountPath, wrapperServerBin),
		"--port", strconv.Itoa(defaultWrapperServerPort), "--",
	}, fnCmd...)

	// Create a pod
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    pe.namespace,
			GenerateName: "krm-fn-",
			// TODO: add labels
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:  "copy-wrapper-server",
					Image: pe.wrapperServerImage,
					Command: []string{
						"sh", "-c",
						fmt.Sprintf("cp /wrapper-server/* %v", volumeMountPath),
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: volumeMountPath,
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "function",
					Image: req.Image,
					Command: []string{
						"sh", "-c",
						strings.Join(cmd, " "),
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									path.Join(volumeMountPath, gRPCProbeBin),
									"-addr", fmt.Sprintf("localhost:%v", strconv.Itoa(defaultWrapperServerPort)),
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: volumeMountPath,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
	err = pe.kubeClient.Create(ctx, pod)
	if err != nil {
		return nil, err
	}
	klog.Infof("created KRM function evaluator pod %v", pod.Name)

	defer func() {
		// TODO: keep the pod around for some time for potential reuse.
		klog.Infof("deleting KRM function evaluator pod %v", pod.Name)
		if err := pe.kubeClient.Delete(ctx, pod); err != nil {
			klog.Warningf("failed to delete KRM function evaluator pod %v: %v", pod.Name, err)
		}
	}()

	// Wait until the pod is Running
	if err = wait.PollImmediate(100*time.Millisecond, 60*time.Second, func() (done bool, err error) {
		err = pe.kubeClient.Get(ctx, client.ObjectKeyFromObject(pod), pod)
		if err != nil {
			return false, err
		}
		if pod.Status.Phase != "Running" {
			return false, nil
		}
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady {
				return cond.Status == corev1.ConditionTrue, nil
			}
		}
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("error when waiting the pod to be ready: %w", err)
	}

	podIP := pod.Status.PodIP
	if podIP == "" {
		return nil, fmt.Errorf("pod did not have podIP")
	}
	address := fmt.Sprintf("%v:%v", podIP, defaultWrapperServerPort)
	klog.Infof("dialing pod function runner %q", address)

	// TODO: pool connections
	cc, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial grpc function evaluator on %q for pod %s/%s: %w", address, pod.Namespace, pod.Name, err)
	}
	defer func() {
		if err := cc.Close(); err != nil {
			klog.Warningf("failed to close grpc connection: %v", err)
		}
	}()

	client := evaluator.NewFunctionEvaluatorClient(cc)
	return client.EvaluateFunction(ctx, req)
}
