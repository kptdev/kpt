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
	"os"
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
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	reclaimAfterAnnotation   = "fn.kpt.dev/reclaim-after"
	fieldManagerName         = "krm-function-runner"

	channelBufferSize = 128
)

type podEvaluator struct {
	requestCh chan<- *clientConnRequest

	podCacheManager *podCacheManager
}

var _ Evaluator = &podEvaluator{}

func NewPodEvaluator(namespace, wrapperServerImage string, interval, ttl time.Duration, podTTLConfig string) (Evaluator, error) {
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
			waitlists:      map[string][]chan<- *clientConnAndError{},

			podManager: &podManager{
				kubeClient:         cl,
				namespace:          namespace,
				wrapperServerImage: wrapperServerImage,
				podReadyCh:         readyCh,
			},
		},
	}
	go pe.podCacheManager.podCacheManager()

	// TODO(mengqiy): add watcher that support reloading the cache when the config file was changed.
	err = pe.podCacheManager.warmupCache(podTTLConfig)
	// If we can't warm up the cache, we can still proceed without it.
	if err != nil {
		klog.Warningf("unable to warm up the pod cache: %w", err)
	}
	return pe, nil
}

func (pe *podEvaluator) EvaluateFunction(ctx context.Context, req *evaluator.EvaluateFunctionRequest) (*evaluator.EvaluateFunctionResponse, error) {
	starttime := time.Now()
	defer func() {
		klog.Infof("evaluating %v in pod took %v", req.Image, time.Now().Sub(starttime))
	}()
	// make a buffer for the channel to prevent unnecessary blocking when the pod cache manager sends it to multiple waiting gorouthine in batch.
	ccChan := make(chan *clientConnAndError, 1)
	// Send a request to request a grpc client.
	pe.requestCh <- &clientConnRequest{
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

	// requestCh is a receive-only channel to receive
	requestCh <-chan *clientConnRequest
	// podReadyCh is a channel to receive the information when a pod is ready.
	podReadyCh <-chan *imagePodAndGRPCClient

	cache     map[string]*podAndGRPCClient
	waitlists map[string][]chan<- *clientConnAndError

	podManager *podManager
}

type clientConnRequest struct {
	image string

	unavavilable     bool
	currClientTarget string

	// grpcConn is a channel that a grpc client should be sent back.
	grpcClientCh chan<- *clientConnAndError
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

func (pcm *podCacheManager) warmupCache(podTTLConfig string) error {
	start := time.Now()
	defer func() {
		klog.Infof("cache warning is completed and it took %v", time.Now().Sub(start))
	}()
	content, err := os.ReadFile(podTTLConfig)
	if err != nil {
		return err
	}
	var podsTTL map[string]string
	err = yaml.Unmarshal(content, &podsTTL)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for fnImage, ttlStr := range podsTTL {
		wg.Add(1)
		go func(img, ttlSt string) {
			klog.Infof("preloading pod cache for function %v with TTL %v", img, ttlSt)
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			ttl, err := time.ParseDuration(ttlSt)
			if err != nil {
				klog.Warningf("unable to parse duration from the config file for function %v: %w", fnImage, err)
				ttl = pcm.podTTL
			}
			// We invoke the function with useGenerateName=false so that the pod name is fixed,
			// since we want to ensure only one pod is created for each function.
			pcm.podManager.getFuncEvalPodClient(ctx, img, ttl, false)
			klog.Infof("preloaded pod cache for function %v", img)
		}(fnImage, ttlStr)
	}
	// Wait for the cache warming to finish before returning.
	wg.Wait()
	return nil
}

func (pcm *podCacheManager) podCacheManager() {
	tick := time.Tick(pcm.gcScanInternal)
	for {
		select {
		case req := <-pcm.requestCh:
			podAndCl, found := pcm.cache[req.image]
			if found && podAndCl != nil {
				// Ensure the pod still exists and is not being deleted before sending the gprc client back to the channel.
				// We can't simply return grpc client from the cache and let evaluator try to connect to the pod.
				// If the pod is deleted by others, it will take ~10 seconds for the evaluator to fail.
				// Wasting 10 second is so much, so we check if the pod still exist first.
				pod := &corev1.Pod{}
				err := pcm.podManager.kubeClient.Get(context.Background(), podAndCl.pod, pod)
				deleteCacheEntry := false
				if err == nil {
					if pod.DeletionTimestamp == nil && net.JoinHostPort(pod.Status.PodIP, defaultWrapperServerPort) == podAndCl.grpcClient.Target() {
						klog.Infof("reusing the connection to pod %v/%v to evaluate %v", pod.Namespace, pod.Name, req.image)
						req.grpcClientCh <- &clientConnAndError{grpcClient: podAndCl.grpcClient}
						go patchPodWithUnixTimeAnnotation(pcm.podManager.kubeClient, podAndCl.pod, pcm.podTTL)
						break
					} else {
						deleteCacheEntry = true
					}
				} else if errors.IsNotFound(err) {
					deleteCacheEntry = true
				}
				// We delete the cache entry if the pod has been deleted or being deleted.
				if deleteCacheEntry {
					delete(pcm.cache, req.image)
				}
			}
			_, found = pcm.waitlists[req.image]
			if !found {
				pcm.waitlists[req.image] = []chan<- *clientConnAndError{}
			}
			list := pcm.waitlists[req.image]
			pcm.waitlists[req.image] = append(list, req.grpcClientCh)
			// We invoke the function with useGenerateName=true to avoid potential name collision, since if pod foo is
			// being deleted and we can't use the same name.
			go pcm.podManager.getFuncEvalPodClient(context.Background(), req.image, pcm.podTTL, true)
		case resp := <-pcm.podReadyCh:
			if resp.err != nil {
				klog.Warningf("received error from the pod manager: %v", resp.err)
			} else {
				pcm.cache[resp.image] = resp.podAndGRPCClient
			}
			channels := pcm.waitlists[resp.image]
			delete(pcm.waitlists, resp.image)
			for i := range channels {
				// The channel has one buffer size, nothing will be blocking.
				channels[i] <- &clientConnAndError{
					grpcClient: resp.grpcClient,
					err:        resp.err,
				}
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
		reclaimAfterStr, found := pod.Annotations[reclaimAfterAnnotation]
		// If a pod doesn't have a last-use annotation, we patch it. This should not happen, but if it happens,
		// we give another TTL before deleting it.
		if !found {
			go patchPodWithUnixTimeAnnotation(pcm.podManager.kubeClient, client.ObjectKeyFromObject(&pod), pcm.podTTL)
			continue
		} else {
			reclaimAfter, err := strconv.ParseInt(reclaimAfterStr, 10, 64)
			// If the annotation is ill-formatted, we patch it with the current time and will try to GC it later.
			// This should not happen, but if it happens, we give another TTL before deleting it.
			if err != nil {
				klog.Warningf("unable to convert the Unix time string to int64: %w", err)
				go patchPodWithUnixTimeAnnotation(pcm.podManager.kubeClient, client.ObjectKeyFromObject(&pod), pcm.podTTL)
				continue
			}
			// If the current time is after the reclaim-ater annotation in the pod, we delete the pod and remove the corresponding cache entry.
			if time.Now().After(time.Unix(reclaimAfter, 0)) {
				podIP := pod.Status.PodIP
				go func(po corev1.Pod) {
					klog.Infof("deleting pod %v/%v", po.Namespace, po.Name)
					err := pcm.podManager.kubeClient.Delete(context.Background(), &po)
					if err != nil {
						klog.Warningf("unable to delete pod %v/%v: %w", po.Namespace, po.Name, err)
					}
				}(podList.Items[i])

				image := pod.Spec.Containers[0].Image
				podAndCl, found := pcm.cache[image]
				if found {
					host, _, err := net.SplitHostPort(podAndCl.grpcClient.Target())
					// If the client target in the cache points to a different pod IP, it means the matching pod is not the current pod.
					// We will keep this cache entry.
					if err == nil && host != podIP {
						continue
					}
					// We delete the cache entry when the IP of the old pod match the client target in the cache
					// or we can't split the host and port in the client target.
					delete(pcm.cache, image)
				}
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

	// podReadyCh is a channel to receive requests to get GRPC client from each function evaluation request handler.
	podReadyCh chan<- *imagePodAndGRPCClient

	// imageMetadataCache is a cache of image name to digestAndEntrypoint.
	// Only podManager is allowed to touch this cache.
	// Its underlying type is map[string]*digestAndEntrypoint.
	imageMetadataCache sync.Map
}

type digestAndEntrypoint struct {
	// digest is a hex string
	digest string
	// entrypoint is the entrypoint of the image
	entrypoint []string
}

func (pm *podManager) getFuncEvalPodClient(ctx context.Context, image string, ttl time.Duration, useGenerateName bool) {
	c, err := func() (*podAndGRPCClient, error) {
		podKey, err := pm.retrieveOrCreatePod(ctx, image, ttl, useGenerateName)
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

// imageEntrypoint get the entrypoint of a container image by looking at its metadata.
func (pm *podManager) imageDigestAndEntrypoint(image string) (*digestAndEntrypoint, error) {
	start := time.Now()
	defer func() {
		klog.Infof("getting image metadata for %v took %v", image, time.Now().Sub(start))
	}()
	var entrypoint []string
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, err
	}
	hash, err := img.Digest()
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
	de := &digestAndEntrypoint{
		digest:     hash.Hex,
		entrypoint: entrypoint,
	}
	pm.imageMetadataCache.Store(image, de)
	return de, nil
}

func (pm *podManager) retrieveOrCreatePod(ctx context.Context, image string, ttl time.Duration, useGenerateName bool) (client.ObjectKey, error) {
	var de *digestAndEntrypoint
	var err error
	val, found := pm.imageMetadataCache.Load(image)
	if !found {
		de, err = pm.imageDigestAndEntrypoint(image)
		if err != nil {
			return client.ObjectKey{}, fmt.Errorf("unable to get the entrypoint for %v: %w", image, err)
		}
	} else {
		de = val.(*digestAndEntrypoint)
	}

	podId, err := podID(image, de.digest)
	if err != nil {
		return client.ObjectKey{}, err
	}

	// Try to retrieve the pod. Lookup the pod by label to see if there is a pod that can be reused.
	// Looking it up locally may not work if there are more than one instance of the function runner,
	// since the pod may be created by one the other instance and the current instance is not aware of it.
	// TODO: It's possible to set up a Watch in the fn runner namespace, and always try to maintain a up-to-date local cache.
	podList := &corev1.PodList{}
	err = pm.kubeClient.List(ctx, podList, client.InNamespace(pm.namespace), client.MatchingLabels(map[string]string{krmFunctionLabel: podId}))
	if err != nil {
		klog.Warningf("error when listing pods for %q: %w", image, err)
	}
	if err == nil && len(podList.Items) > 0 {
		// TODO: maybe we should randomly pick one that is no being deleted.
		for _, pod := range podList.Items {
			if pod.DeletionTimestamp == nil {
				klog.Infof("retrieved function evaluator pod %v/%v for %q", pod.Namespace, pod.Name, image)
				return client.ObjectKeyFromObject(&pod), nil
			}
		}
	}

	cmd := append([]string{
		path.Join(volumeMountPath, wrapperServerBin),
		"--port", defaultWrapperServerPort, "--",
	}, de.entrypoint...)

	// Create a pod
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pm.namespace,
			Annotations: map[string]string{
				reclaimAfterAnnotation: fmt.Sprintf("%v", time.Now().Add(ttl).Unix()),
			},
			// The function runner can use the label to retrieve the pod. Label is function name + part of its digest.
			// If a function has more than one tags pointing to the same digest, we can reuse the same pod.
			// TODO: controller-runtime provides field indexer, we can potentially use it to index spec.containers[*].image field.
			Labels: map[string]string{
				krmFunctionLabel: podId,
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
	if useGenerateName {
		pod.GenerateName = podId + "-"
		err = pm.kubeClient.Create(ctx, pod, client.FieldOwner(fieldManagerName))
		if err != nil {
			return client.ObjectKey{}, fmt.Errorf("unable to apply the pod: %w", err)
		}
	} else {
		pod.Name = podId
		err = pm.kubeClient.Patch(ctx, pod, client.Apply, client.FieldOwner(fieldManagerName))
		if err != nil {
			return client.ObjectKey{}, fmt.Errorf("unable to apply the pod: %w", err)
		}
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

func patchPodWithUnixTimeAnnotation(cl client.Client, podKey client.ObjectKey, ttl time.Duration) {
	patch := []byte(fmt.Sprintf(`{"metadata":{"annotations":{"%v": "%v"}}}`, reclaimAfterAnnotation, time.Now().Add(ttl).Unix()))
	pod := &corev1.Pod{}
	pod.Namespace = podKey.Namespace
	pod.Name = podKey.Name
	if err := cl.Patch(context.Background(), pod, client.RawPatch(types.MergePatchType, patch)); err != nil {
		klog.Warningf("unable to patch last-use annotation for pod %v/%v: %w", podKey.Namespace, podKey.Name, err)
	}
}

func podID(image, hash string) (string, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return "", fmt.Errorf("unable to parse image reference %v: %w", image, err)
	}

	// repoName will be something like gcr.io/kpt-fn/set-namespace
	repoName := ref.Context().Name()
	parts := strings.Split(repoName, "/")
	name := strings.ReplaceAll(parts[len(parts)-1], "_", "-")
	return fmt.Sprintf("%v-%v", name, hash[:8]), nil
}
