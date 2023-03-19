// Copyright 2023 Google LLC
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

package controllers

import (
	"context"
	"time"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	e2eclusters "github.com/GoogleContainerTools/kpt/rollouts/e2e/clusters"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Rollout", func() {
	var targets []e2eclusters.Config
	var targetClusterSetup e2eclusters.ClusterSetup
	var RolloutName = "test-rollout"
	var RolloutNamespace = "default"

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("A non progressive Rollout", func() {
		// setup target clusters
		BeforeEach(func() {
			var err error
			targets = []e2eclusters.Config{
				{
					Prefix: "e2e-sjc-",
					Count:  1,
					Labels: map[string]string{
						"city": "sjc",
					},
				},
				{
					Prefix: "e2e-sfo-",
					Count:  1,
					Labels: map[string]string{
						"city": "sfo",
					},
				},
			}
			targetClusterSetup, err = e2eclusters.GetClusterSetup(tt, k8sClient, targets...)
			Expect(err).NotTo(HaveOccurred())

			err = targetClusterSetup.PrepareAndWait(context.TODO(), 5*time.Minute)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("tearing down the target clusters")
			_ = targetClusterSetup.Cleanup(context.TODO())
		})
		It("Should deploy package to only matched target clusters", func() {
			By("By creating a new Rollout")
			ctx := context.Background()
			rollout := &gitopsv1alpha1.Rollout{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "gitops.kpt.dev/v1alpha1",
					Kind:       "Rollout",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      RolloutName,
					Namespace: RolloutNamespace,
				},
				Spec: gitopsv1alpha1.RolloutSpec{
					Description: "Test Rollout",
					Packages: gitopsv1alpha1.PackagesConfig{
						SourceType: gitopsv1alpha1.GitHub,
						GitHub: gitopsv1alpha1.GitHubSource{
							Selector: gitopsv1alpha1.GitHubSelector{
								Org:       "droot",
								Repo:      "store",
								Directory: "namespaces",
								Revision:  "v3",
							},
						},
					},
					Clusters: gitopsv1alpha1.ClusterDiscovery{
						SourceType: gitopsv1alpha1.KindCluster,
					},
					Targets: gitopsv1alpha1.ClusterTargetSelector{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"city": "sjc",
							},
						},
					},
					PackageToTargetMatcher: gitopsv1alpha1.PackageToClusterMatcher{
						Type: gitopsv1alpha1.MatchAllClusters,
					},
					SyncTemplate: &gitopsv1alpha1.SyncTemplate{
						Type: gitopsv1alpha1.TemplateTypeRootSync,
					},
					Strategy: gitopsv1alpha1.RolloutStrategy{
						Type: gitopsv1alpha1.AllAtOnce,
					},
				},
			}
			Expect(k8sClient.Create(ctx, rollout)).Should(Succeed())
			/*
				After creating this Rollout, let's check that the Rollout's Spec fields match what we passed in.
				Note that, because the k8s apiserver may not have finished creating a Rollout after our `Create()` call from earlier, we will use Gomega’s Eventually() testing function instead of Expect() to give the apiserver an opportunity to finish creating our CronJob.
				`Eventually()` will repeatedly run the function provided as an argument every interval seconds until
				(a) the function’s output matches what’s expected in the subsequent `Should()` call, or
				(b) the number of attempts * interval period exceed the provided timeout value.
				In the examples below, timeout and interval are Go Duration values of our choosing.
			*/

			rolloutLookupKey := types.NamespacedName{Name: RolloutName, Namespace: RolloutNamespace}
			createdRollout := &gitopsv1alpha1.Rollout{}

			// We'll need to retry getting this newly created Rollout, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, rolloutLookupKey, createdRollout)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdRollout.Spec.Description).Should(Equal("Test Rollout"))

			remoteSyncKey := types.NamespacedName{Name: "github-589324850-namespaces-e2e-sjc-0", Namespace: RolloutNamespace}
			remoteSync := &gitopsv1alpha1.RemoteSync{}

			// We should eventually have a remotesync object created corresponding to a target cluster
			Eventually(func() bool {
				err := k8sClient.Get(ctx, remoteSyncKey, remoteSync)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(remoteSync.Spec.Template.Spec.Git.Repo).Should(Equal("https://github.com/droot/store.git"))
			Expect(remoteSync.Spec.Template.Spec.Git.Revision).Should(Equal("v3"))

			// We should eventually have the rollout completed
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, rolloutLookupKey, createdRollout)
				return createdRollout.Status.Overall == "Completed"
			}, 1*time.Minute, interval).Should(BeTrue())

			Expect(createdRollout.Status.ClusterStatuses).Should(HaveLen(1))
			Expect(createdRollout.Status.ClusterStatuses).Should(ContainElement(gitopsv1alpha1.ClusterStatus{
				Name: "e2e-sjc-0",
				PackageStatus: gitopsv1alpha1.PackageStatus{
					PackageID:  "github-589324850-namespaces-e2e-sjc-0",
					Status:     "Synced",
					SyncStatus: "Synced",
				},
			}))
			forground := metav1.DeletePropagationForeground
			Expect(k8sClient.Delete(context.TODO(), createdRollout, &client.DeleteOptions{PropagationPolicy: &forground})).NotTo(HaveOccurred())
			// We should wait for the rollout to be deleted
			Eventually(func() bool {
				err := k8sClient.Get(ctx, rolloutLookupKey, createdRollout)
				return client.IgnoreNotFound(err) == nil
			}, 1*time.Minute, interval).Should(BeTrue())
		})
	})

	Context("A progressive Rollout", func() {
		// setup target clusters
		BeforeEach(func() {
			var err error
			targets = []e2eclusters.Config{
				{
					Prefix: "e2e-sjcc-",
					Count:  1,
					Labels: map[string]string{
						"city": "sjcc",
					},
				},
				{
					Prefix: "e2e-sfoo-",
					Count:  1,
					Labels: map[string]string{
						"city": "sfoo",
					},
				},
			}
			targetClusterSetup, err = e2eclusters.GetClusterSetup(tt, k8sClient, targets...)
			Expect(err).NotTo(HaveOccurred())

			err = targetClusterSetup.PrepareAndWait(context.TODO(), 5*time.Minute)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("tearing down the target clusters")
			_ = targetClusterSetup.Cleanup(context.TODO())
		})

		It("Should deploy package to matched target clusters", func() {
			By("By creating a new Rollout")
			ctx := context.Background()
			RolloutName = "test-city-rollout"

			/*
							apiVersion: gitops.kpt.dev/v1alpha1
				kind: ProgressiveRolloutStrategy
				metadata:
				  name: stores-rollout-strategy
				spec:
				  waves:
				    - name: GA-stores
				      targets:
				        selector:
				          matchLabels:
				            state: ga
				      maxConcurrent: 1
				    - name: NY-stores
				      targets:
				        selector:
				          matchLabels:
				            state: ny
				      maxConcurrent: 1
				    - name: CA-stores
				      targets:
				        selector:
				          matchLabels:
				            state: ca
				      maxConcurrent: 2
			*/
			RolloutStrategyName := "city-wide-rollout"
			progressiveRolloutStrategy := &gitopsv1alpha1.ProgressiveRolloutStrategy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "gitops.kpt.dev/v1alpha1",
					Kind:       "ProgressiveRolloutStrategy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      RolloutStrategyName,
					Namespace: RolloutNamespace,
				},
				Spec: gitopsv1alpha1.ProgressiveRolloutStrategySpec{
					Waves: []gitopsv1alpha1.Wave{
						{
							Name: "sjc-stores",
							Targets: gitopsv1alpha1.ClusterTargetSelector{
								Selector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"city": "sjcc",
									},
								},
							},
							MaxConcurrent: 1,
						},
						{
							Name: "sfo-stores",
							Targets: gitopsv1alpha1.ClusterTargetSelector{
								Selector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"city": "sfoo",
									},
								},
							},
							MaxConcurrent: 1,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, progressiveRolloutStrategy)).Should(Succeed())
			strategyLookupKey := types.NamespacedName{Name: RolloutStrategyName, Namespace: RolloutNamespace}
			createdRolloutStrategy := &gitopsv1alpha1.ProgressiveRolloutStrategy{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, strategyLookupKey, createdRolloutStrategy)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			rollout := &gitopsv1alpha1.Rollout{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "gitops.kpt.dev/v1alpha1",
					Kind:       "Rollout",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      RolloutName,
					Namespace: RolloutNamespace,
				},
				Spec: gitopsv1alpha1.RolloutSpec{
					Description: "Test Rollout",
					Packages: gitopsv1alpha1.PackagesConfig{
						SourceType: gitopsv1alpha1.GitHub,
						GitHub: gitopsv1alpha1.GitHubSource{
							Selector: gitopsv1alpha1.GitHubSelector{
								Org:       "droot",
								Repo:      "store",
								Directory: "namespaces",
								Revision:  "v3",
							},
						},
					},
					Clusters: gitopsv1alpha1.ClusterDiscovery{
						SourceType: gitopsv1alpha1.KindCluster,
					},
					Targets: gitopsv1alpha1.ClusterTargetSelector{
						Selector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "city",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"sjcc", "sfoo"},
								},
							},
						},
					},
					PackageToTargetMatcher: gitopsv1alpha1.PackageToClusterMatcher{
						Type: gitopsv1alpha1.MatchAllClusters,
					},
					SyncTemplate: &gitopsv1alpha1.SyncTemplate{
						Type: gitopsv1alpha1.TemplateTypeRootSync,
					},
					Strategy: gitopsv1alpha1.RolloutStrategy{
						Type: gitopsv1alpha1.Progressive,
						Progressive: &gitopsv1alpha1.StrategyProgressive{
							Name:      RolloutStrategyName,
							Namespace: RolloutNamespace,
							PauseAfterWave: gitopsv1alpha1.PauseAfterWave{
								WaveName: "sfo-stores",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, rollout)).Should(Succeed())
			/*
				After creating this Rollout, let's check that the Rollout's Spec fields match what we passed in.
				Note that, because the k8s apiserver may not have finished creating a Rollout after our `Create()` call from earlier, we will use Gomega’s Eventually() testing function instead of Expect() to give the apiserver an opportunity to finish creating our CronJob.
				`Eventually()` will repeatedly run the function provided as an argument every interval seconds until
				(a) the function’s output matches what’s expected in the subsequent `Should()` call, or
				(b) the number of attempts * interval period exceed the provided timeout value.
				In the examples below, timeout and interval are Go Duration values of our choosing.
			*/

			rolloutLookupKey := types.NamespacedName{Name: RolloutName, Namespace: RolloutNamespace}
			createdRollout := &gitopsv1alpha1.Rollout{}

			// We'll need to retry getting this newly created Rollout, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, rolloutLookupKey, createdRollout)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdRollout.Spec.Description).Should(Equal("Test Rollout"))

			remoteSyncKey := types.NamespacedName{Name: "github-589324850-namespaces-e2e-sjcc-0", Namespace: RolloutNamespace}
			remoteSync := &gitopsv1alpha1.RemoteSync{}

			// We should eventually have a remotesync object created corresponding to a target cluster
			Eventually(func() bool {
				err := k8sClient.Get(ctx, remoteSyncKey, remoteSync)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(remoteSync.Spec.Template.Spec.Git.Repo).Should(Equal("https://github.com/droot/store.git"))
			Expect(remoteSync.Spec.Template.Spec.Git.Revision).Should(Equal("v3"))

			// We should eventually have the rollout completed
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, rolloutLookupKey, createdRollout)
				return createdRollout.Status.Overall == "Completed"
			}, 2*time.Minute, interval).Should(BeTrue())

			Expect(createdRollout.Status.ClusterStatuses).Should(HaveLen(2))
			Expect(createdRollout.Status.ClusterStatuses).Should(ContainElement(gitopsv1alpha1.ClusterStatus{
				Name: "e2e-sjcc-0",
				PackageStatus: gitopsv1alpha1.PackageStatus{
					PackageID:  "github-589324850-namespaces-e2e-sjcc-0",
					Status:     "Synced",
					SyncStatus: "Synced",
				},
			}))
			Expect(createdRollout.Status.ClusterStatuses).Should(ContainElement(gitopsv1alpha1.ClusterStatus{
				Name: "e2e-sfoo-0",
				PackageStatus: gitopsv1alpha1.PackageStatus{
					PackageID:  "github-589324850-namespaces-e2e-sfoo-0",
					Status:     "Synced",
					SyncStatus: "Synced",
				},
			}))
		})
	})
})
