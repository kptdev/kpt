// Copyright 2023 The kpt Authors
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
	"fmt"
	"testing"
	"time"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	e2eclusters "github.com/GoogleContainerTools/kpt/rollouts/e2e/clusters"
	"github.com/GoogleContainerTools/kpt/rollouts/pkg/clusterstore"
	"github.com/GoogleContainerTools/kpt/rollouts/pkg/packagediscovery"
	"github.com/google/go-github/v48/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
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

			Expect(targetClusterSetup.PrepareAndWait(context.TODO(), 5*time.Minute)).To(Succeed())
		})

		AfterEach(func() {
			By("tearing down the target clusters")
			Expect(targetClusterSetup.Cleanup(context.TODO())).To(Succeed())
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
			Expect(k8sClient.Create(ctx, rollout)).To(Succeed())
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
			Expect(remoteSync.Spec.Template.Spec.Git.Repo).To(Equal("https://github.com/droot/store.git"))
			Expect(remoteSync.Spec.Template.Spec.Git.Revision).To(Equal("v3"))

			// We should eventually have the rollout completed
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, rolloutLookupKey, createdRollout)
				return createdRollout.Status.Overall == "Completed"
			}, 1*time.Minute, interval).Should(BeTrue())

			Expect(createdRollout.Status.ClusterStatuses).To(HaveLen(1))
			Expect(createdRollout.Status.ClusterStatuses).To(ContainElement(gitopsv1alpha1.ClusterStatus{
				Name: "e2e-sjc-0",
				PackageStatus: gitopsv1alpha1.PackageStatus{
					PackageID:  "github-589324850-namespaces-e2e-sjc-0",
					Status:     "Synced",
					SyncStatus: "Synced",
				},
			}))
			forground := metav1.DeletePropagationForeground
			Expect(k8sClient.Delete(context.TODO(), createdRollout, &client.DeleteOptions{PropagationPolicy: &forground})).To(Succeed())
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

			Expect(targetClusterSetup.PrepareAndWait(context.TODO(), 5*time.Minute)).To(Succeed())
		})

		AfterEach(func() {
			By("tearing down the target clusters")
			Expect(targetClusterSetup.Cleanup(context.TODO())).To(Succeed())
		})

		It("Should deploy package to matched target clusters", func() {
			By("By creating a new Rollout")
			ctx := context.Background()
			RolloutName = "test-city-rollout"

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

			Expect(k8sClient.Create(ctx, progressiveRolloutStrategy)).To(Succeed())
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
								WaveName: "sjc-stores", // pause at the first wave
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, rollout)).To(Succeed())

			rolloutLookupKey := types.NamespacedName{Name: RolloutName, Namespace: RolloutNamespace}
			createdRollout := &gitopsv1alpha1.Rollout{}

			// We'll need to retry getting this newly created Rollout, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, rolloutLookupKey, createdRollout)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdRollout.Spec.Description).To(Equal("Test Rollout"))

			remoteSyncKey := types.NamespacedName{Name: "github-589324850-namespaces-e2e-sjcc-0", Namespace: RolloutNamespace}
			remoteSync := &gitopsv1alpha1.RemoteSync{}

			// We should eventually have a remotesync object created corresponding to a target cluster
			Eventually(func() bool {
				err := k8sClient.Get(ctx, remoteSyncKey, remoteSync)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(remoteSync.Spec.Template.Spec.Git.Repo).To(Equal("https://github.com/droot/store.git"))
			Expect(remoteSync.Spec.Template.Spec.Git.Revision).To(Equal("v3"))

			// expect rollout to be in waiting state after completion of the first wave
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, rolloutLookupKey, createdRollout)
				return createdRollout.Status.Overall == "Waiting"
			}, 2*time.Minute, interval).Should(BeTrue())

			// advance the rollout to next wave
			createdRollout.Spec.Strategy.Progressive.PauseAfterWave.WaveName = "sfo-stores"
			Expect(k8sClient.Update(ctx, createdRollout)).To(Succeed())

			// expect rollout to be in completed now
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, rolloutLookupKey, createdRollout)
				return createdRollout.Status.Overall == "Completed"
			}, 2*time.Minute, interval).Should(BeTrue())

			Expect(createdRollout.Status.ClusterStatuses).To(HaveLen(2))
			Expect(createdRollout.Status.ClusterStatuses).To(
				ContainElements(
					gitopsv1alpha1.ClusterStatus{
						Name: "e2e-sjcc-0",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-589324850-namespaces-e2e-sjcc-0",
							Status:     "Synced",
							SyncStatus: "Synced",
						},
					},
					gitopsv1alpha1.ClusterStatus{
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

func TestReconcileRollout(t *testing.T) {
	// create an arbitrary package
	testURL := "https://test.com/git"
	discoveredPackage := packagediscovery.DiscoveredPackage{
		Directory: "dir",
		Revision:  "v0",
		Branch:    "main",
		GitHubRepo: &github.Repository{
			CloneURL: &testURL,
		},
	}

	t.Run("StrategyAllAtOnce", func(t *testing.T) {
		// This tests that if the strategy is AllAtOnce, all clusters are synced
		// at the same time.
		rollout := &gitopsv1alpha1.Rollout{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Rollout",
				APIVersion: "gitops.kpt.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "sample",
			},

			Spec: gitopsv1alpha1.RolloutSpec{
				PackageToTargetMatcher: gitopsv1alpha1.PackageToClusterMatcher{
					Type: "AllClusters",
				},
				Strategy: gitopsv1alpha1.RolloutStrategy{
					Type: "AllAtOnce",
				},
				Targets: gitopsv1alpha1.ClusterTargetSelector{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
			},
		}

		// create 3 target clusters with arbitrary names
		targetClusters := make([]clusterstore.Cluster, 3)
		for i := 0; i < 3; i++ {
			targetClusters[i] = clusterstore.Cluster{
				Ref:    gitopsv1alpha1.ClusterRef{Name: fmt.Sprintf("foo/%d", i)},
				Labels: map[string]string{"foo": "bar"},
			}
		}

		fc := newFakeRemoteSyncClient()
		reconciler := (&RolloutReconciler{Client: fc})

		strategy, err := reconciler.getStrategy(context.Background(), rollout)
		require.NoError(t, err)

		_, waveStatus, err := reconciler.reconcileRollout(
			context.Background(),
			rollout,
			strategy,
			targetClusters,
			[]packagediscovery.DiscoveredPackage{discoveredPackage},
		)

		require.NoError(t, err)
		require.Equal(t, []string{
			"listing objects",
			"getting object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-1\"",
			"getting object named \"github-0-dir-2\"",
			"creating object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-0\"",
			"creating object named \"github-0-dir-1\"",
			"getting object named \"github-0-dir-1\"",
			"creating object named \"github-0-dir-2\"",
			"getting object named \"github-0-dir-2\"",
		}, fc.actions)
		require.Equal(t, 3, len(fc.remotesyncs))
		require.Equal(t, []gitopsv1alpha1.WaveStatus{
			{
				Name:   "",
				Status: "Progressing",
				Paused: false,
				ClusterStatuses: []gitopsv1alpha1.ClusterStatus{
					{
						Name: "foo/0",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-0",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
					{
						Name: "foo/1",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-1",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
					{
						Name: "foo/2",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-2",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
				},
			},
		}, waveStatus)
	})

	t.Run("UpdateAndDeleteRemoteSyncs", func(t *testing.T) {
		// This tests that if there are existing RemoteSyncs that need to be updated
		// or deleted, reconcileRollout updates/deletes them as needed.
		rollout := &gitopsv1alpha1.Rollout{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Rollout",
				APIVersion: "gitops.kpt.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "sample",
			},

			Spec: gitopsv1alpha1.RolloutSpec{
				PackageToTargetMatcher: gitopsv1alpha1.PackageToClusterMatcher{
					Type: "AllClusters",
				},
				Strategy: gitopsv1alpha1.RolloutStrategy{
					Type: "AllAtOnce",
				},
				Targets: gitopsv1alpha1.ClusterTargetSelector{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
			},
		}

		// create 2 target clusters with arbitrary names
		targetClusters := make([]clusterstore.Cluster, 2)
		for i := 0; i < 2; i++ {
			targetClusters[i] = clusterstore.Cluster{
				Ref:    gitopsv1alpha1.ClusterRef{Name: fmt.Sprintf("foo/%d", i)},
				Labels: map[string]string{"foo": "bar"},
			}
		}

		fc := newFakeRemoteSyncClient()
		reconciler := (&RolloutReconciler{Client: fc})

		strategy, err := reconciler.getStrategy(context.Background(), rollout)
		require.NoError(t, err)

		// create two existing remotesyncs, one that will need to be updated and another
		// that will need to be deleted
		fc.remotesyncs[types.NamespacedName{Name: "to-be-deleted"}] = gitopsv1alpha1.RemoteSync{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Rollout",
				APIVersion: "gitops.kpt.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "to-be-deleted",
			},
		}
		fc.remotesyncs[types.NamespacedName{Name: "github-0-dir-0"}] = gitopsv1alpha1.RemoteSync{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Rollout",
				APIVersion: "gitops.kpt.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "github-0-dir-0",
			},
			Spec: gitopsv1alpha1.RemoteSyncSpec{
				ClusterRef: gitopsv1alpha1.ClusterRef{Name: "clusterRef"},
			},
		}

		_, _, err = reconciler.reconcileRollout(
			context.Background(),
			rollout,
			strategy,
			targetClusters,
			[]packagediscovery.DiscoveredPackage{discoveredPackage},
		)

		require.NoError(t, err)
		require.Equal(t, []string{
			"listing objects",
			"getting object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-1\"",
			"creating object named \"github-0-dir-1\"",
			"getting object named \"github-0-dir-1\"",
			"updating object named \"github-0-dir-0\"",
			"deleting object named \"to-be-deleted\"",
		}, fc.actions)
		require.Equal(t, 2, len(fc.remotesyncs))
	})

	t.Run("RootSyncLabelsAndAnnotations", func(t *testing.T) {
		// Tests the labels/annotations object metadata get properly propagated
		// to the remotesync objects.
		labels := map[string]string{"labelKey": "labelValue"}
		annotations := map[string]string{"annoKey": "annoValue"}

		rollout := &gitopsv1alpha1.Rollout{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Rollout",
				APIVersion: "gitops.kpt.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "sample",
			},

			Spec: gitopsv1alpha1.RolloutSpec{
				PackageToTargetMatcher: gitopsv1alpha1.PackageToClusterMatcher{
					Type: "AllClusters",
				},
				Strategy: gitopsv1alpha1.RolloutStrategy{
					Type: "AllAtOnce",
				},
				Targets: gitopsv1alpha1.ClusterTargetSelector{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
				SyncTemplate: &gitopsv1alpha1.SyncTemplate{
					Type: gitopsv1alpha1.TemplateTypeRootSync,
					RootSync: &gitopsv1alpha1.RootSyncTemplate{
						Metadata: &gitopsv1alpha1.Metadata{
							Labels:      labels,
							Annotations: annotations,
						},
					},
				},
			},
		}

		// create 3 target clusters with arbitrary names
		targetClusters := make([]clusterstore.Cluster, 3)
		for i := 0; i < 3; i++ {
			targetClusters[i] = clusterstore.Cluster{
				Ref:    gitopsv1alpha1.ClusterRef{Name: fmt.Sprintf("foo/%d", i)},
				Labels: map[string]string{"foo": "bar"},
			}
		}

		fc := newFakeRemoteSyncClient()
		reconciler := (&RolloutReconciler{Client: fc})

		strategy, err := reconciler.getStrategy(context.Background(), rollout)
		require.NoError(t, err)

		_, _, err = reconciler.reconcileRollout(
			context.Background(),
			rollout,
			strategy,
			targetClusters,
			[]packagediscovery.DiscoveredPackage{discoveredPackage},
		)

		require.NoError(t, err)
		require.Equal(t, 3, len(fc.remotesyncs))
		for _, rs := range fc.remotesyncs {
			require.Equal(t, rs.Spec.Template.Metadata.Labels, labels)
			require.Equal(t, rs.Spec.Template.Metadata.Annotations, annotations)
		}
	})

	t.Run("CELMatcher", func(t *testing.T) {
		// tests the CEL expression matcher
		rollout := &gitopsv1alpha1.Rollout{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Rollout",
				APIVersion: "gitops.kpt.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "sample",
			},

			Spec: gitopsv1alpha1.RolloutSpec{
				PackageToTargetMatcher: gitopsv1alpha1.PackageToClusterMatcher{
					Type:            "Custom",
					MatchExpression: "cluster.name == rolloutPackage.directory",
				},
				Strategy: gitopsv1alpha1.RolloutStrategy{
					Type: "AllAtOnce",
				},
				Targets: gitopsv1alpha1.ClusterTargetSelector{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
			},
		}

		// create 3 target clusters and 3 rollouts packages with arbitrary names
		targetClusters := make([]clusterstore.Cluster, 3)
		discoveredPackages := make([]packagediscovery.DiscoveredPackage, 3)
		for i := 0; i < 3; i++ {
			clusterName := fmt.Sprintf("foo/%d", i)
			targetClusters[i] = clusterstore.Cluster{
				Ref:    gitopsv1alpha1.ClusterRef{Name: clusterName},
				Labels: map[string]string{"foo": "bar"},
			}
			discoveredPackages[i] = packagediscovery.DiscoveredPackage{
				Directory: clusterName,
				Revision:  "v0",
				Branch:    "main",
				GitHubRepo: &github.Repository{
					CloneURL: &testURL,
					Name:     &testURL,
				},
			}
		}

		fc := newFakeRemoteSyncClient()
		reconciler := (&RolloutReconciler{Client: fc})

		strategy, err := reconciler.getStrategy(context.Background(), rollout)
		require.NoError(t, err)

		_, waveStatus, err := reconciler.reconcileRollout(
			context.Background(),
			rollout,
			strategy,
			targetClusters,
			discoveredPackages,
		)

		require.NoError(t, err)
		require.Equal(t, []string{
			"listing objects",
			"getting object named \"github-0-foo/0-0\"",
			"getting object named \"github-0-foo/1-1\"",
			"getting object named \"github-0-foo/2-2\"",
			"creating object named \"github-0-foo/0-0\"",
			"getting object named \"github-0-foo/0-0\"",
			"creating object named \"github-0-foo/1-1\"",
			"getting object named \"github-0-foo/1-1\"",
			"creating object named \"github-0-foo/2-2\"",
			"getting object named \"github-0-foo/2-2\"",
		}, fc.actions)

		// ensure that each cluster got the correct package according to the CEL expression
		require.Equal(t, 3, len(fc.remotesyncs))
		for _, rs := range fc.remotesyncs {
			require.Equal(t, rs.Spec.ClusterRef.Name, rs.Spec.Template.Spec.Git.Dir)
		}

		require.Equal(t, []gitopsv1alpha1.WaveStatus{
			{
				Name:   "",
				Status: "Progressing",
				Paused: false,
				ClusterStatuses: []gitopsv1alpha1.ClusterStatus{
					{
						Name: "foo/0",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-foo/0-0",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
					{
						Name: "foo/1",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-foo/1-1",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
					{
						Name: "foo/2",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-foo/2-2",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
				},
			},
		}, waveStatus)
	})

	t.Run("StrategyRollingUpdate", func(t *testing.T) {
		// This tests that if "maxConcurrent" is set to 1 with RollingUpdate strategy,
		// only one cluster is synced at  a time. This test calls `reconcileRollout` twice.
		// After the first call, we check that only the first cluster starts progressing while
		// the second is  waiting. Then, we manually update the sync status of the first
		// remotesync to "Synced", call `reconcileRollout` again, and verify that the second
		// cluster starts progressing.
		rollout := &gitopsv1alpha1.Rollout{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Rollout",
				APIVersion: "gitops.kpt.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "sample",
			},

			Spec: gitopsv1alpha1.RolloutSpec{
				PackageToTargetMatcher: gitopsv1alpha1.PackageToClusterMatcher{
					Type: "AllClusters",
				},
				Strategy: gitopsv1alpha1.RolloutStrategy{
					Type: "RollingUpdate",
					RollingUpdate: &gitopsv1alpha1.StrategyRollingUpdate{
						MaxConcurrent: 1,
					},
				},
				Targets: gitopsv1alpha1.ClusterTargetSelector{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
			},
		}

		// create 2 target clusters with arbitrary names
		targetClusters := make([]clusterstore.Cluster, 2)
		for i := 0; i < 2; i++ {
			targetClusters[i] = clusterstore.Cluster{
				Ref:    gitopsv1alpha1.ClusterRef{Name: fmt.Sprintf("foo/%d", i)},
				Labels: map[string]string{"foo": "bar"},
			}
		}

		fc := newFakeRemoteSyncClient()
		reconciler := (&RolloutReconciler{Client: fc})

		strategy, err := reconciler.getStrategy(context.Background(), rollout)
		require.NoError(t, err)

		// first call to reconcileRollout - only one cluster should start progressing
		_, waveStatus, err := reconciler.reconcileRollout(
			context.Background(),
			rollout,
			strategy,
			targetClusters,
			[]packagediscovery.DiscoveredPackage{discoveredPackage},
		)

		require.NoError(t, err)
		require.Equal(t, []string{
			"listing objects",
			"getting object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-1\"",
			"creating object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-0\"",
		}, fc.actions)
		require.Equal(t, 1, len(fc.remotesyncs))
		require.Equal(t, []gitopsv1alpha1.WaveStatus{
			{
				Name:   "",
				Status: "Progressing",
				Paused: false,
				ClusterStatuses: []gitopsv1alpha1.ClusterStatus{
					{
						Name: "foo/0",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-0",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
					{
						Name: "foo/1",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-1",
							SyncStatus: "",
							Status:     "Waiting",
						},
					},
				},
			},
		}, waveStatus)

		// reset actions and set sync status of remote sync to "synced"
		fc.actions = nil
		require.NoError(t, fc.setSyncStatus(types.NamespacedName{Name: "github-0-dir-0", Namespace: ""}, "Synced"))

		// second call to reconcileRollout - the second cluster should now progress
		_, waveStatus, err = reconciler.reconcileRollout(
			context.Background(),
			rollout,
			strategy,
			targetClusters,
			[]packagediscovery.DiscoveredPackage{discoveredPackage},
		)

		require.NoError(t, err)
		require.Equal(t, []string{
			"listing objects",
			"getting object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-1\"",
			"creating object named \"github-0-dir-1\"",
			"getting object named \"github-0-dir-1\"",
		}, fc.actions)
		require.Equal(t, 2, len(fc.remotesyncs))
		require.Equal(t, []gitopsv1alpha1.WaveStatus{
			{
				Name:   "",
				Status: "Progressing",
				Paused: false,
				ClusterStatuses: []gitopsv1alpha1.ClusterStatus{
					{
						Name: "foo/0",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-0",
							SyncStatus: "Synced",
							Status:     "Synced",
						},
					},
					{
						Name: "foo/1",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-1",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
				},
			},
		}, waveStatus)
	})

	t.Run("StrategyProgressiveWithMaxConcurrent", func(t *testing.T) {
		// This test is identical to the above `StrategyRollingUpdate`, except that it
		// sets maxConcurrent to 1 in a ProgressiveRolloutStrategy.
		rollout := &gitopsv1alpha1.Rollout{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Rollout",
				APIVersion: "gitops.kpt.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "sample",
			},

			Spec: gitopsv1alpha1.RolloutSpec{
				PackageToTargetMatcher: gitopsv1alpha1.PackageToClusterMatcher{
					Type: "AllClusters",
				},
				Strategy: gitopsv1alpha1.RolloutStrategy{
					Type: "Progressive",
					Progressive: &gitopsv1alpha1.StrategyProgressive{
						Name: "sample",
					},
				},
				Targets: gitopsv1alpha1.ClusterTargetSelector{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
			},
		}

		// create 2 target clusters with arbitrary names
		targetClusters := make([]clusterstore.Cluster, 2)
		for i := 0; i < 2; i++ {
			targetClusters[i] = clusterstore.Cluster{
				Ref:    gitopsv1alpha1.ClusterRef{Name: fmt.Sprintf("foo/%d", i)},
				Labels: map[string]string{"foo": "bar"},
			}
		}

		fc := newFakeRemoteSyncClient()
		reconciler := (&RolloutReconciler{Client: fc})

		strategy := &gitopsv1alpha1.ProgressiveRolloutStrategy{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ProgressiveRolloutStrategy",
				APIVersion: "gitops.kpt.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "sample",
			},

			Spec: gitopsv1alpha1.ProgressiveRolloutStrategySpec{
				Waves: []gitopsv1alpha1.Wave{
					{
						Name:          "wave-1",
						MaxConcurrent: 1,

						// selects all clusters
						Targets: gitopsv1alpha1.ClusterTargetSelector{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"foo": "bar"},
							},
						},
					},
				},
			},
		}

		// first call to reconcileRollout - only one cluster should start progressing
		_, waveStatus, err := reconciler.reconcileRollout(
			context.Background(),
			rollout,
			strategy,
			targetClusters,
			[]packagediscovery.DiscoveredPackage{discoveredPackage},
		)

		require.NoError(t, err)
		require.Equal(t, []string{
			"listing objects",
			"getting object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-1\"",
			"creating object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-0\"",
		}, fc.actions)
		require.Equal(t, 1, len(fc.remotesyncs))
		require.Equal(t, []gitopsv1alpha1.WaveStatus{
			{
				Name:   "wave-1",
				Status: "Progressing",
				Paused: false,
				ClusterStatuses: []gitopsv1alpha1.ClusterStatus{
					{
						Name: "foo/0",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-0",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
					{
						Name: "foo/1",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-1",
							SyncStatus: "",
							Status:     "Waiting",
						},
					},
				},
			},
		}, waveStatus)

		// reset actions and set sync status of remote sync to "synced"
		fc.actions = nil
		require.NoError(t, fc.setSyncStatus(types.NamespacedName{Name: "github-0-dir-0", Namespace: ""}, "Synced"))

		// second call to reconcileRollout - the second cluster should now progress
		_, waveStatus, err = reconciler.reconcileRollout(
			context.Background(),
			rollout,
			strategy,
			targetClusters,
			[]packagediscovery.DiscoveredPackage{discoveredPackage},
		)

		require.NoError(t, err)
		require.Equal(t, []string{
			"listing objects",
			"getting object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-1\"",
			"creating object named \"github-0-dir-1\"",
			"getting object named \"github-0-dir-1\"",
		}, fc.actions)
		require.Equal(t, 2, len(fc.remotesyncs))
		require.Equal(t, []gitopsv1alpha1.WaveStatus{
			{
				Name:   "wave-1",
				Status: "Progressing",
				Paused: false,
				ClusterStatuses: []gitopsv1alpha1.ClusterStatus{
					{
						Name: "foo/0",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-0",
							SyncStatus: "Synced",
							Status:     "Synced",
						},
					},
					{
						Name: "foo/1",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-1",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
				},
			},
		}, waveStatus)
	})

	t.Run("StrategyProgressiveWithWaves", func(t *testing.T) {
		// This test checks the functionality of pauseAfterWave for the ProgressiveRolloutStrategy.
		// It calls reconcileRollout for the first wave, verifying that only those
		// clusters in the first wave are being synced. Then we manually the sync status of the
		// remotesyncs from the first wave to "synced", and call reconcileRollout again to ensure
		// that the second wave is still paused. Then we update the PauseAfterWave value in the
		// Rollout object (equivalent to running `kpt alpha rollouts advance`), and call
		// reconcileRollout again, verifying that the clusters in the second wave start syncing.
		rollout := &gitopsv1alpha1.Rollout{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Rollout",
				APIVersion: "gitops.kpt.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "sample",
			},

			Spec: gitopsv1alpha1.RolloutSpec{
				PackageToTargetMatcher: gitopsv1alpha1.PackageToClusterMatcher{
					Type: "AllClusters",
				},
				Strategy: gitopsv1alpha1.RolloutStrategy{
					Type: "Progressive",
					Progressive: &gitopsv1alpha1.StrategyProgressive{
						Name:           "sample",
						PauseAfterWave: gitopsv1alpha1.PauseAfterWave{WaveName: "wave-1"},
					},
				},
				Targets: gitopsv1alpha1.ClusterTargetSelector{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
			},
		}

		// create 3 target clusters with arbitrary names, two for the first wave
		// and one for the second.
		targetClusters := make([]clusterstore.Cluster, 3)
		for i := 0; i < 2; i++ {
			targetClusters[i] = clusterstore.Cluster{
				Ref:    gitopsv1alpha1.ClusterRef{Name: fmt.Sprintf("foo/%d", i)},
				Labels: map[string]string{"wave": "one"},
			}
		}
		targetClusters[2] = clusterstore.Cluster{
			Ref:    gitopsv1alpha1.ClusterRef{Name: fmt.Sprintf("foo/%d", 2)},
			Labels: map[string]string{"wave": "two"},
		}

		fc := newFakeRemoteSyncClient()
		reconciler := (&RolloutReconciler{Client: fc})

		strategy := &gitopsv1alpha1.ProgressiveRolloutStrategy{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ProgressiveRolloutStrategy",
				APIVersion: "gitops.kpt.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "sample",
			},

			Spec: gitopsv1alpha1.ProgressiveRolloutStrategySpec{
				Waves: []gitopsv1alpha1.Wave{
					{
						Name:          "wave-1",
						MaxConcurrent: 3,
						Targets: gitopsv1alpha1.ClusterTargetSelector{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"wave": "one"},
							},
						},
					},
					{
						Name:          "wave-2",
						MaxConcurrent: 3,
						Targets: gitopsv1alpha1.ClusterTargetSelector{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"wave": "two"},
							},
						},
					},
				},
			},
		}

		// first call to reconcileRollout - only two clusters (first wave) should start progressing
		_, waveStatus, err := reconciler.reconcileRollout(
			context.Background(),
			rollout,
			strategy,
			targetClusters,
			[]packagediscovery.DiscoveredPackage{discoveredPackage},
		)

		require.NoError(t, err)
		require.Equal(t, []string{
			"listing objects",
			"getting object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-1\"",
			"getting object named \"github-0-dir-2\"",
			"creating object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-0\"",
			"creating object named \"github-0-dir-1\"",
			"getting object named \"github-0-dir-1\"",
		}, fc.actions)
		require.Equal(t, 2, len(fc.remotesyncs))
		require.Equal(t, []gitopsv1alpha1.WaveStatus{
			{
				Name:   "wave-1",
				Status: "Progressing",
				Paused: false,
				ClusterStatuses: []gitopsv1alpha1.ClusterStatus{
					{
						Name: "foo/0",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-0",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
					{
						Name: "foo/1",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-1",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
				},
			},
			{
				Name:   "wave-2",
				Status: "Waiting",
				Paused: true,
				ClusterStatuses: []gitopsv1alpha1.ClusterStatus{
					{
						Name: "foo/2",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-2",
							SyncStatus: "",
							Status:     "Waiting (Upcoming Wave)",
						},
					},
				},
			},
		}, waveStatus)

		// reset actions and set sync status of remote syncs to "synced"
		fc.actions = nil
		require.NoError(t, fc.setSyncStatus(types.NamespacedName{Name: "github-0-dir-0", Namespace: ""}, "Synced"))
		require.NoError(t, fc.setSyncStatus(types.NamespacedName{Name: "github-0-dir-1", Namespace: ""}, "Synced"))

		// second call to reconcileRollout - the first wave should be completed and the second wave should still be paused
		_, waveStatus, err = reconciler.reconcileRollout(
			context.Background(),
			rollout,
			strategy,
			targetClusters,
			[]packagediscovery.DiscoveredPackage{discoveredPackage},
		)

		require.NoError(t, err)
		require.Equal(t, []string{
			"listing objects",
			"getting object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-1\"",
			"getting object named \"github-0-dir-2\"",
		}, fc.actions)
		require.Equal(t, 2, len(fc.remotesyncs))
		require.Equal(t, []gitopsv1alpha1.WaveStatus{
			{
				Name:   "wave-1",
				Status: "Completed",
				Paused: false,
				ClusterStatuses: []gitopsv1alpha1.ClusterStatus{
					{
						Name: "foo/0",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-0",
							SyncStatus: "Synced",
							Status:     "Synced",
						},
					},
					{
						Name: "foo/1",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-1",
							SyncStatus: "Synced",
							Status:     "Synced",
						},
					},
				},
			},
			{
				Name:   "wave-2",
				Status: "Waiting",
				Paused: true,
				ClusterStatuses: []gitopsv1alpha1.ClusterStatus{
					{
						Name: "foo/2",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-2",
							SyncStatus: "",
							Status:     "Waiting (Upcoming Wave)",
						},
					},
				},
			},
		}, waveStatus)

		// reset actions and advance to the second wave
		fc.actions = nil
		rollout.Spec.Strategy.Progressive.PauseAfterWave = gitopsv1alpha1.PauseAfterWave{WaveName: "wave-2"}

		// third call to reconcileRollout - the third cluster (second wave) should now progress
		_, waveStatus, err = reconciler.reconcileRollout(
			context.Background(),
			rollout,
			strategy,
			targetClusters,
			[]packagediscovery.DiscoveredPackage{discoveredPackage},
		)

		require.NoError(t, err)
		require.Equal(t, []string{
			"listing objects",
			"getting object named \"github-0-dir-0\"",
			"getting object named \"github-0-dir-1\"",
			"getting object named \"github-0-dir-2\"",
			"creating object named \"github-0-dir-2\"",
			"getting object named \"github-0-dir-2\"",
		}, fc.actions)
		require.Equal(t, 3, len(fc.remotesyncs))
		require.Equal(t, []gitopsv1alpha1.WaveStatus{
			{
				Name:   "wave-1",
				Status: "Completed",
				Paused: false,
				ClusterStatuses: []gitopsv1alpha1.ClusterStatus{
					{
						Name: "foo/0",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-0",
							SyncStatus: "Synced",
							Status:     "Synced",
						},
					},
					{
						Name: "foo/1",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-1",
							SyncStatus: "Synced",
							Status:     "Synced",
						},
					},
				},
			},
			{
				Name:   "wave-2",
				Status: "Progressing",
				Paused: false,
				ClusterStatuses: []gitopsv1alpha1.ClusterStatus{
					{
						Name: "foo/2",
						PackageStatus: gitopsv1alpha1.PackageStatus{
							PackageID:  "github-0-dir-2",
							SyncStatus: "",
							Status:     "Progressing",
						},
					},
				},
			},
		}, waveStatus)
	})
}
