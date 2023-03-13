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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Rollout controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		RolloutName      = "test-rollout"
		RolloutNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	// TODO(droot): Improve the textual description
	Context("When creating Rollout", func() {
		It("Should succeed", func() {
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
								Org:      "droot",
								Repo:     "oahu",
								Revision: "main",
							},
						},
					},
					Clusters: gitopsv1alpha1.ClusterDiscovery{
						SourceType: gitopsv1alpha1.KCC,
					},
					Targets: gitopsv1alpha1.ClusterTargetSelector{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"env": "dev",
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
		})
	})
})
