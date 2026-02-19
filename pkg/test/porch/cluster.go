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

package porch

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	TestGitServerImage = "test-git-server"
)

func GetGitServerImageName(t *testing.T) string {
	cmd := exec.Command("kubectl", "get", "pods", "--selector=app=porch-server", "--namespace=porch-system",
		"--output=jsonpath={.items[*].spec.containers[*].image}")

	var stderr bytes.Buffer
	var stdout bytes.Buffer

	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("Error when getting Porch image version: %v: %s", err, stderr.String())
	}

	out := stdout.String()
	t.Logf("Porch image query output: %s", out)

	lines := strings.Split(out, "\n")
	if len(lines) == 0 {
		t.Fatalf("kubectl get pods didn't return any images: %s", out)
	}
	image := strings.TrimSpace(lines[0])
	if image == "" {
		t.Fatalf("Cannot determine Porch server image: output was %q", out)
	}
	return InferGitServerImage(image)
}

func InferGitServerImage(porchImage string) string {
	slash := strings.LastIndex(porchImage, "/")
	repo := porchImage[:slash+1]
	image := porchImage[slash+1:]
	colon := strings.LastIndex(image, ":")
	tag := image[colon+1:]

	return repo + TestGitServerImage + ":" + tag
}

func KubectlApply(t *testing.T, config string) {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(config)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kubectl apply failed: %v\ninput: %s\n\noutput:%s", err, config, string(out))
	}
	t.Logf("kubectl apply\n%s\noutput:\n%s", config, string(out))
}

func KubectlWaitForDeployment(t *testing.T, namespace, name string) {
	args := []string{"rollout", "status", "deployment", "--namespace", namespace, name}
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kubectl %s failed: %v\noutput:\n%s", strings.Join(args, " "), err, string(out))
	}
	t.Logf("kubectl %s:\n%s", strings.Join(args, " "), string(out))
}

func KubectlWaitForService(t *testing.T, namespace, name string) {
	args := []string{"get", "endpoints", "--namespace", namespace, name, "--output=jsonpath='{.subsets[*].addresses[*].ip}'"}

	giveUp := time.Now().Add(1 * time.Minute)
	for {
		cmd := exec.Command("kubectl", args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		s := stdout.String()
		if err == nil && len(s) > 0 { // Endpoint has an IP address assigned
			t.Logf("Endpoints: %q", s)
			break
		}

		if time.Now().After(giveUp) {
			var msg string
			if err != nil {
				msg = err.Error()
			}
			t.Fatalf("Service endpoint %s/%s not ready on time. Giving up: %s", namespace, name, msg)
		}

		time.Sleep(5 * time.Second)
	}
}

// Kubernetes DNS needs time to propagate the updated address
// Wait until we can register the repository and list its contents.
func KubectlWaitForGitDNS(t *testing.T, gitServerURL string) {
	const name = "test-git-dns-resolve"

	KubectlCreateNamespace(t, name)
	defer KubectlDeleteNamespace(t, name)

	// We expect repos to automatically be created (albeit empty)
	repoURL := gitServerURL + "/" + name

	cmd := exec.Command("kpt", "alpha", "repo", "register", "--namespace", name, "--name", name, repoURL)
	t.Logf("running command %v", strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to register probe repository: %v\n%s", err, string(out))
	}

	// Based on experience, DNS seems to get updated inside the cluster within
	// few seconds. We will wait about a minute.
	// If this turns out to be an issue, we will sidestep DNS and use the Endpoints
	// IP address directly.
	giveUp := time.Now().Add(1 * time.Minute)
	for {
		cmd := exec.Command("kpt", "alpha", "rpkg", "get", "--namespace", name)
		t.Logf("running command %v", strings.Join(cmd.Args, " "))
		out, err := cmd.CombinedOutput()
		t.Logf("output: %v", string(out))

		if err == nil {
			break
		}

		if time.Now().After(giveUp) {
			t.Fatalf("Git service DNS resolution failed: %v", err)
		}

		time.Sleep(5 * time.Second)
	}
}

func KubectlCreateNamespace(t *testing.T, name string) {
	cmd := exec.Command("kubectl", "create", "namespace", name)
	t.Logf("running command %v", strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create namespace %q: %v\n%s", name, err, string(out))
	}
	t.Logf("output: %v", string(out))
}

func KubectlDeleteNamespace(t *testing.T, name string) {
	cmd := exec.Command("kubectl", "delete", "namespace", name)
	t.Logf("running command %v", strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Failed to delete namespace %q: %v\n%s", name, err, string(out))
	}
	t.Logf("output: %v", string(out))
}

func RegisterRepository(t *testing.T, repoURL, namespace, name string) {
	cmd := exec.Command("kpt", "alpha", "repo", "register", "--namespace", namespace, "--name", name, repoURL)
	t.Logf("running command %v", strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to register repository %q: %v\n%s", repoURL, err, string(out))
	}
	t.Logf("output: %v", string(out))
}
