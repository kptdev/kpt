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

// Package kptops contains implementations of kpt operations
package kptops

import (
	"fmt"
	"strings"

	kptfilev1 "github.com/kptdev/kpt/pkg/api/kptfile/v1"
	"github.com/kptdev/kpt/pkg/kptfile/kptfileutil"
)

func UpdateUpstream(kptfileContents string, name string, upstream kptfilev1.Upstream, lock kptfilev1.Locator) (string, error) {
	// Normalize the repository URL and directory path
	normalizeGitFields(&upstream)
	normalizeGitLockFields(&lock) // Use separate function for lock

	return updateKptfileContentsPreservingFormat(kptfileContents, func(kptfile *kptfilev1.KptFile) {
		kptfile.UpstreamLock = &lock
		kptfile.Upstream = &upstream
		if name != "" {
			kptfile.Name = name
		}
	})
}

func UpdateName(kptfileContents string, name string) (string, error) {
	return updateKptfileContentsPreservingFormat(kptfileContents, func(kptfile *kptfilev1.KptFile) {
		kptfile.Name = name
	})
}

func updateKptfileContentsPreservingFormat(kptfileContents string, mutator func(*kptfilev1.KptFile)) (string, error) {
	out, err := kptfileutil.UpdateKptfileContent(kptfileContents, mutator)
	if err != nil {
		return "", fmt.Errorf("cannot update Kptfile: %w", err)
	}

	return out, nil
}

func UpdateKptfileUpstream(name string, contents map[string]string, upstream kptfilev1.Upstream, lock kptfilev1.Locator) error {
	kptfile, found := contents[kptfilev1.KptFileName]
	if !found {
		return fmt.Errorf("package %q is missing Kptfile", name)
	}

	kptfile, err := UpdateUpstream(kptfile, name, upstream, lock)
	if err != nil {
		return fmt.Errorf("failed to update package upstream: %w", err)
	}

	contents[kptfilev1.KptFileName] = kptfile
	return nil
}

func UpdateKptfileName(name string, contents map[string]string) error {
	kptfile, found := contents[kptfilev1.KptFileName]
	if !found {
		return fmt.Errorf("package %q is missing Kptfile", name)
	}

	kptfile, err := UpdateName(kptfile, name)
	if err != nil {
		return fmt.Errorf("failed to update package upstream: %w", err)
	}

	contents[kptfilev1.KptFileName] = kptfile
	return nil
}

// normalizeGitFields ensures consistent formatting of git repository URLs and directory paths
func normalizeGitFields(u *kptfilev1.Upstream) {
	if u.Git != nil {
		// Ensure .git suffix is present
		if !strings.HasSuffix(u.Git.Repo, ".git") {
			u.Git.Repo += ".git"
		}

		// Ensure directory doesn't start with a slash
		u.Git.Directory = strings.TrimPrefix(u.Git.Directory, "/")
	}
}

// normalizeGitLockFields ensures consistent formatting for Locator git fields
func normalizeGitLockFields(l *kptfilev1.Locator) {
	if l.Git != nil {
		// Ensure .git suffix is present
		if !strings.HasSuffix(l.Git.Repo, ".git") {
			l.Git.Repo += ".git"
		}

		// Ensure directory doesn't start with a slash
		l.Git.Directory = strings.TrimPrefix(l.Git.Directory, "/")
	}
}
