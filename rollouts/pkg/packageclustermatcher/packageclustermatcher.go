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

package packageclustermatcher

import (
	"fmt"

	gkeclusterapis "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/container/v1beta1"
	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/GoogleContainerTools/kpt/rollouts/pkg/packagediscovery"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
)

type PackageClusterMatcher struct {
	clusters []gkeclusterapis.ContainerCluster
	packages []packagediscovery.DiscoveredPackage
}

type ClusterPackages struct {
	Cluster  gkeclusterapis.ContainerCluster
	Packages []packagediscovery.DiscoveredPackage
}

func NewPackageClusterMatcher(clusters []gkeclusterapis.ContainerCluster, packages []packagediscovery.DiscoveredPackage) *PackageClusterMatcher {
	return &PackageClusterMatcher{
		clusters: clusters,
		packages: packages,
	}
}

func (m *PackageClusterMatcher) GetClusterPackages(matcher gitopsv1alpha1.PackageToClusterMatcher) ([]ClusterPackages, error) {
	clusters := m.clusters
	packages := m.packages

	allClusterPackages := []ClusterPackages{}

	for _, cluster := range clusters {
		matchedPackages := []packagediscovery.DiscoveredPackage{}

		celCluster := map[string]interface{}{
			"name":   cluster.ObjectMeta.Name,
			"labels": cluster.ObjectMeta.Labels,
		}

		for _, discoveredPackage := range packages {
			celPackage := map[string]interface{}{
				"org":       discoveredPackage.Org,
				"repo":      discoveredPackage.Repo,
				"directory": discoveredPackage.Directory,
			}

			isMatch, err := isPackageClusterMatch(matcher.MatchExpression, celCluster, celPackage)
			if err != nil {
				return nil, fmt.Errorf("unable to execute package cluster matcher expression: %w", err)
			}

			if isMatch {
				matchedPackages = append(matchedPackages, discoveredPackage)
			}
		}

		clusterPackages := ClusterPackages{
			Cluster:  cluster,
			Packages: matchedPackages,
		}

		allClusterPackages = append(allClusterPackages, clusterPackages)
	}

	return allClusterPackages, nil
}

func isPackageClusterMatch(expr string, cluster, rolloutPackage map[string]interface{}) (bool, error) {
	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("cluster", decls.Dyn),
			decls.NewVar("rolloutPackage", decls.Dyn),
		))
	if err != nil {
		return false, err
	}

	p, issue := env.Parse(expr)
	if issue != nil && issue.Err() != nil {
		return false, issue.Err()
	}

	c, issue := env.Check(p)
	if issue != nil && issue.Err() != nil {
		return false, issue.Err()
	}

	prg, err := env.Program(c)
	if err != nil {
		return false, err
	}

	out, _, err := prg.Eval(map[string]interface{}{
		"cluster":        cluster,
		"rolloutPackage": rolloutPackage,
	})

	return out.Value().(bool), err
}
