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
	"net"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	defaultWrapperServerPort = "9446"
	volumeName               = "wrapper-server-tools"
	volumeMountPath          = "/wrapper-server-tools"
	wrapperServerBin         = "wrapper-server"
	gRPCProbeBin             = "grpc-health-probe"
	krmFunctionLabel         = "fn.kpt.dev/image"
	lastUseTimeAnnotation    = "fn.kpt.dev/last-use"

	channelBufferSize = 128
)

type podEvaluator struct {
	requestCh chan *clientConnRequest

	podCacheManager *podCacheManager
}

var _ Evaluator = &podEvaluator{}

func NewPodEvaluator(namespace, wrapperServerImage string, interval, ttl time.Duration) (Evaluator, error) {
	restCfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}
	cl, err := client.New(restCfg, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	reqCh := make(chan *clientConnRequest, channelBufferSize)
	readyCh := make(chan *imagePodAndGRPCClient, channelBufferSize)

	pe := &podEvaluator{
		requestCh: reqCh,
		podCacheManager: &podCacheManager{
			gcScanInternal: interval,
			podTTL:         ttl,
			requestCh:      reqCh,
			podReadyCh:     readyCh,
			cache:          map[string]*podAndGRPCClient{},
			waitlists:      map[string][]chan *clientConnAndError{},

			podManager: &podManager{
				kubeClient:         cl,
				namespace:          namespace,
				wrapperServerImage: wrapperServerImage,
				podReadyCh:         readyCh,
			},
		},
	}
	go pe.podCacheManager.podCacheManager()
	return pe, nil
}

func (pe *podEvaluator) EvaluateFunction(ctx context.Context, req *evaluator.EvaluateFunctionRequest) (*evaluator.EvaluateFunctionResponse, error) {
	starttime := time.Now()
	defer func() {
		endtime := time.Now()
		klog.Infof("evaluating %v in pod takes %v", req.Image, endtime.Sub(starttime))
	}()
	// make a buffer for the channel to prevent unnecessary blocking when the pod cache manager sends it to multiple waiting gorouthine in batch.
	ccChan := make(chan *clientConnAndError, 1)
	// Send a request to request a grpc client.
	pe.podCacheManager.requestCh <- &clientConnRequest{
		ctx:          ctx,
		image:        req.Image,
		grpcClientCh: ccChan,
	}

	// Waiting for the client from the channel.
	cc := <-ccChan
	if cc.err != nil {
		return nil, fmt.Errorf("unable to get the grpc client to the pod for %v: %w", req.Image, cc.err)
	}

	resp, err := evaluator.NewFunctionEvaluatorClient(cc.grpcClient).EvaluateFunction(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("unable to evaluate %v with pod evaluator: %w", req.Image, err)
	}
	return resp, nil
}

type podCacheManager struct {
	gcScanInternal time.Duration
	podTTL         time.Duration

	// requestCh is a channel to send the request to cache manager. The cache manager will send back the grpc client in the embedded channel.
	requestCh chan *clientConnRequest
	// podReadyCh is a channel to receive the information when a pod is ready.
	podReadyCh chan *imagePodAndGRPCClient

	cache     map[string]*podAndGRPCClient
	waitlists map[string][]chan *clientConnAndError

	podManager *podManager
}

type clientConnRequest struct {
	ctx   context.Context
	image string

	unavavilable     bool
	currClientTarget string

	// grpcConn is a channel that a grpc client should be sent back.
	grpcClientCh chan *clientConnAndError
}

type clientConnAndError struct {
	grpcClient *grpc.ClientConn
	err        error
}

type podAndGRPCClient struct {
	grpcClient *grpc.ClientConn
	pod        client.ObjectKey
}

type imagePodAndGRPCClient struct {
	image string
	*podAndGRPCClient
	err error
}

func (pcm *podCacheManager) podCacheManager() {
	tick := time.Tick(pcm.gcScanInternal)
	for {
		select {
		case req := <-pcm.requestCh:
			podAndCl, found := pcm.cache[req.image]
			if found && podAndCl != nil {
				// Ensure the pod still exists and is not being deleted before sending the gprc client back to the channel.
				// We can't simplly return grpc client from the cache and let evaluator try to connect to the pod.
				// If the pod is deleted by others, it will take ~10 seconds for the evaluator to fail.
				// Wasting 10 second is so much, so we check if the pod still exist first.
				pod := &corev1.Pod{}
				err := pcm.podManager.kubeClient.Get(req.ctx, podAndCl.pod, pod)
				if err == nil && pod.DeletionTimestamp == nil {
					klog.Infof("reusing the connection to pod %v/%v to evaluate %v", pod.Namespace, pod.Name, req.image)
					req.grpcClientCh <- &clientConnAndError{grpcClient: podAndCl.grpcClient}
					go patchPodWithUnixTimeAnnotation(pcm.podManager.kubeClient, podAndCl.pod)
					break
				}
			}
			_, found = pcm.waitlists[req.image]
			if !found {
				pcm.waitlists[req.image] = []chan *clientConnAndError{}
			}
			list := pcm.waitlists[req.image]
			list = append(list, req.grpcClientCh)
			pcm.waitlists[req.image] = list
			go pcm.podManager.getFuncEvalPodClient(req.ctx, req.image)
		case resp := <-pcm.podReadyCh:
			if resp.err == nil {
				pcm.cache[resp.image] = resp.podAndGRPCClient
				channels := pcm.waitlists[resp.image]
				delete(pcm.waitlists, resp.image)
				for i := range channels {
					// The channel has one buffer size, nothing will be blocking.
					channels[i] <- &clientConnAndError{grpcClient: resp.grpcClient}
				}
			} else {
				klog.Warningf("received error from the pod manager: %v", resp.err)
			}
		case <-tick:
			// synchronous GC
			pcm.garbageCollector()
		}
	}
}

// TODO: We can use Watch + periodically reconciliation to manage the pods,
// the pod evaluator will become a controller.
func (pcm *podCacheManager) garbageCollector() {
	var err error
	podList := &corev1.PodList{}
	err = pcm.podManager.kubeClient.List(context.Background(), podList, client.InNamespace(pcm.podManager.namespace))
	if err != nil {
		klog.Warningf("unable to list pods in namespace %v: %w", pcm.podManager.namespace, err)
		return
	}
	for i, pod := range podList.Items {
		// If a pod is being deleted, skip it.
		if pod.DeletionTimestamp != nil {
			continue
		}
		lastUse, found := pod.Annotations[lastUseTimeAnnotation]
		// If a pod doesn't have a last-use annotation, we patch it.
		if !found {
			go patchPodWithUnixTimeAnnotation(pcm.podManager.kubeClient, client.ObjectKeyFromObject(&pod))
			continue
		} else {
			lu, err := strconv.ParseInt(lastUse, 10, 64)
			// If the annotation is ill-formatted, we patch it with the current time and will try to GC it later.
			if err != nil {
				klog.Warningf("unable to convert the Unix time string to int64: %w", err)
				go patchPodWithUnixTimeAnnotation(pcm.podManager.kubeClient, client.ObjectKeyFromObject(&pod))
				continue
			}
			if time.Unix(lu, 0).Add(pcm.podTTL).Before(time.Now()) {
				image := pod.Spec.Containers[0].Image
				podAndCl, found := pcm.cache[image]
				if found {
					// We delete the cache entry when its grpc client points to the old pod IP.
					host, _, err := net.SplitHostPort(podAndCl.grpcClient.Target())
					if err != nil {
						klog.Warningf("unable to split the GRPC dialer target to host and port : %w", err)
						continue
					}
					if host == pod.Status.PodIP {
						delete(pcm.cache, image)
					}
				}

				go func(po corev1.Pod) {
					klog.Infof("deleting pod %v/%v", po.Namespace, po.Name)
					err := pcm.podManager.kubeClient.Delete(context.Background(), &po)
					if err != nil {
						klog.Warningf("unable to delete pod %v/%v: %w", po.Namespace, po.Name, err)
					}
				}(podList.Items[i])
			}
		}
	}
}

type podManager struct {
	// kubeClient is the kubernetes client
	kubeClient client.Client
	// namespace holds the namespace where the executors run
	namespace string
	// wrapperServerImage is the image name of the wrapper server
	wrapperServerImage string

	// podReadyCh is a channel to send the grpc client information.
	// podCacheManager receives from this channel.
	podReadyCh chan *imagePodAndGRPCClient

	// entrypointCache is a cache of image name to entrypoint.
	// Only podManager is allowed to touch this cache.
	// Its underlying type is map[string][]string.
	entrypointCache sync.Map
}

func (pm *podManager) getFuncEvalPodClient(ctx context.Context, image string) {
	c, err := func() (*podAndGRPCClient, error) {
		podKey, err := pm.retrieveOrCreatePod(ctx, image)
		if err != nil {
			return nil, err
		}
		podIP, err := pm.podIpIfRunningAndReady(ctx, podKey)
		if err != nil {
			return nil, err
		}
		if podIP == "" {
			return nil, fmt.Errorf("pod %s/%s did not have podIP", podKey.Namespace, podKey.Name)
		}
		address := net.JoinHostPort(podIP, defaultWrapperServerPort)
		cc, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("failed to dial grpc function evaluator on %q for pod %s/%s: %w", address, podKey.Namespace, podKey.Name, err)
		}
		return &podAndGRPCClient{
			pod:        podKey,
			grpcClient: cc,
		}, err
	}()
	pm.podReadyCh <- &imagePodAndGRPCClient{
		image:            image,
		podAndGRPCClient: c,
		err:              err,
	}
}

func (pm *podManager) imageEntrypoint(image string) ([]string, error) {
	// Create pod otherwise.
	var entrypoint []string
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, err
	}
	cf, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}

	cfg := cf.Config
	// TODO: to handle all scenario, we should follow https://docs.docker.com/engine/reference/builder/#understand-how-cmd-and-entrypoint-interact.
	if len(cfg.Entrypoint) != 0 {
		entrypoint = cfg.Entrypoint
	} else {
		entrypoint = cfg.Cmd
	}
	pm.entrypointCache.Store(image, entrypoint)
	return entrypoint, nil
}

func (pm *podManager) retrieveOrCreatePod(ctx context.Context, image string) (client.ObjectKey, error) {
	// Try to retrieve the pod first.
	// Lookup the pod by label to see if there is a pod that can be reused.
	// Looking it up locally may not work if there are more than one instance of the function runner,
	// since the pod may be created by one the other instance and the current instance is not aware of it.
	// TODO: It's possible to set up a Watch in the fn runner namespace, and always try to maintain a up-to-date local cache.
	podList := &corev1.PodList{}
	err := pm.kubeClient.List(ctx, podList, client.InNamespace(pm.namespace), client.MatchingLabels(map[string]string{krmFunctionLabel: transformImageName(image)}))
	if err == nil && len(podList.Items) > 0 {
		// TODO: maybe we should randomly pick one that is no being deleted.
		for _, pod := range podList.Items {
			if pod.DeletionTimestamp == nil {
				klog.Infof("retrieved function evaluator pod %v/%v for %q", pod.Namespace, pod.Name, image)
				return client.ObjectKeyFromObject(&pod), nil
			}
		}
	}

	var entrypoint []string
	val, found := pm.entrypointCache.Load(image)
	if !found {
		entrypoint, err = pm.imageEntrypoint(image)
		if err != nil {
			return client.ObjectKey{}, fmt.Errorf("unable to get the entrypoint for %v: %w", image, err)
		}
	} else {
		entrypoint = val.([]string)
	}

	cmd := append([]string{
		path.Join(volumeMountPath, wrapperServerBin),
		"--port", defaultWrapperServerPort, "--",
	}, entrypoint...)

	// Create a pod
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    pm.namespace,
			GenerateName: "krm-fn-",
			Annotations: map[string]string{
				lastUseTimeAnnotation: fmt.Sprintf("%v", time.Now().Unix()),
			},
			// The function runner can use the label to retrieve the pod
			// TODO: controller-runtime provides field indexer, we can potentially use it to index spec.containers[*].image field.
			Labels: map[string]string{
				krmFunctionLabel: transformImageName(image),
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:  "copy-wrapper-server",
					Image: pm.wrapperServerImage,
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
					Image: image,
					Command: []string{
						"sh", "-c",
						strings.Join(cmd, " "),
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									path.Join(volumeMountPath, gRPCProbeBin),
									"-addr", net.JoinHostPort("localhost", defaultWrapperServerPort),
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
	err = pm.kubeClient.Create(ctx, pod)
	if err != nil {
		return client.ObjectKey{}, err
	}
	klog.Infof("created KRM function evaluator pod %v/%v for %q", pod.Namespace, pod.Name, image)
	return client.ObjectKeyFromObject(pod), nil
}

// podIpIfRunningAndReady waits for the pod to be running and ready and returns the pod IP and a potential error.
func (pm *podManager) podIpIfRunningAndReady(ctx context.Context, podKey client.ObjectKey) (ip string, e error) {
	var pod corev1.Pod
	// Wait until the pod is Running
	if e := wait.PollImmediate(100*time.Millisecond, 60*time.Second, func() (done bool, err error) {
		err = pm.kubeClient.Get(ctx, podKey, &pod)
		if err != nil {
			return false, err
		}
		if pod.Status.Phase != "Running" {
			return false, nil
		}
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	}); e != nil {
		return "", fmt.Errorf("error when waiting the pod to be ready: %w", e)
	}
	return pod.Status.PodIP, nil
}

func patchPodWithUnixTimeAnnotation(cl client.Client, podKey client.ObjectKey) {
	patch := []byte(fmt.Sprintf(`{"metadata":{"annotations":{"%v": "%v"}}}`, lastUseTimeAnnotation, time.Now().Unix()))
	pod := &corev1.Pod{}
	pod.Namespace = podKey.Namespace
	pod.Name = podKey.Name
	if err := cl.Patch(context.Background(), pod, client.RawPatch(types.MergePatchType, patch)); err != nil {
		klog.Warningf("unable to patch last-use annotation for pod %v/%v: %w", podKey.Namespace, podKey.Name, err)
	}
}

func transformImageName(image string) string {
	return strings.ReplaceAll(strings.ReplaceAll(image, "/", "__"), ":", "___")
}
