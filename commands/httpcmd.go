// Copyright 2019 Google LLC
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

package commands

import (
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/cmd/kubectl/kubectlcobra"
	"sigs.k8s.io/kustomize/cmd/resource/status"
)

func GetHTTPCommand(name string) *cobra.Command {
	http := &cobra.Command{
		Use:   "http",
		Short: "Apply and make Resource requests to clusters",
	}
	http.AddCommand(status.StatusCommand())
	http.AddCommand(kubectlcobra.GetCommand(nil).Commands()...)
	return http
}
