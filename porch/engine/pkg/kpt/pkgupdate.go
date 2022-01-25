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

package kpt

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	kptlib "github.com/GoogleContainerTools/kpt/pkg/kptlib"
	"k8s.io/klog/v2"
)

// PkgUpdateOpts are options for invoking kpt PkgUpdate
type PkgUpdateOpts struct {
	Strategy string
}

// PkgUpdate is a wrapper around `kpt pkg update`, running it against the package in packageDir
func PkgUpdate(ctx context.Context, ref string, packageDir string, opts PkgUpdateOpts) error {
	// TODO: Printer should be a logr
	pr := kptlib.NewPrinter(os.Stdout, os.Stderr)
	ctx = kptlib.WithPrinterContext(ctx, pr)

	// This code is based on the kpt pkg update code.

	fsys := os.DirFS(packageDir)

	f, err := fsys.Open("Kptfile")
	if err != nil {
		return fmt.Errorf("error opening kptfile: %w", err)
	}
	defer f.Close()

	kf, err := kptlib.ParseKptFile(f)
	if err != nil {
		return fmt.Errorf("error parsing kptfile: %w", err)
	}

	if kf.Upstream == nil || kf.Upstream.Git == nil {
		return fmt.Errorf("package must have an upstream reference") //errors.E(op, u.Pkg.UniquePath,
		// fmt.Errorf("package must have an upstream reference"))
	}
	// originalRootKfRef := rootKf.Upstream.Git.Ref
	if ref != "" {
		kf.Upstream.Git.Ref = ref
	}
	// if u.Strategy != "" {
	// 	rootKf.Upstream.UpdateStrategy = u.Strategy
	// }
	if err = kptfileutil.WriteFile(packageDir, kf); err != nil {
		return err // errors.E(op, u.Pkg.UniquePath, err)
	}

	var updated *kptlib.FetchResults
	// var updatedDigest string
	// var updatedDir string
	var originDir string
	switch kf.Upstream.Type {
	case kptfilev1.GitOrigin:
		g := kf.Upstream.Git
		upstream := &kptlib.GitRepoSpec{OrgRepo: g.Repo, Path: g.Directory, Ref: g.Ref}
		klog.Infof("Fetching upstream from %s@%s\n", upstream.OrgRepo, upstream.Ref)
		// pr.Printf("Fetching upstream from %s@%s\n", kf.Upstream.Git.Repo, kf.Upstream.Git.Ref)
		// if err := fetch.ClonerUsingGitExec(ctx, updated); err != nil {
		// 	return errors.E(op, p.UniquePath, err)
		// }
		fetched, err := kptlib.Fetch(ctx, upstream)
		if err != nil {
			return err //errors.E(op, p.UniquePath, err)
		}
		updated = fetched
		defer os.RemoveAll(updated.AbsPath())
		// updatedDir = updated.AbsPath()

		// var origin repoClone
		if kf.UpstreamLock != nil {
			gLock := kf.UpstreamLock.Git
			originRepoSpec := &kptlib.GitRepoSpec{OrgRepo: gLock.Repo, Path: gLock.Directory, Ref: gLock.Commit}
			klog.Infof("Fetching origin from %s@%s\n", originRepoSpec.OrgRepo, originRepoSpec.Ref)
			// pr.Printf("Fetching origin from %s@%s\n", kf.Upstream.Git.Repo, kf.Upstream.Git.Ref)
			// if err := fetch.ClonerUsingGitExec(ctx, originRepoSpec); err != nil {
			// 	return errors.E(op, p.UniquePath, err)
			// }
			fetched, err := kptlib.Fetch(ctx, originRepoSpec)
			if err != nil {
				return err //errors.E(op, p.UniquePath, err)
			}
			originDir = fetched.AbsPath()
		} else {
			dir, err := ioutil.TempDir("", "kpt-empty-")
			if err != nil {
				return fmt.Errorf("failed to create tempdir: %w", err)
			}
			originDir = dir
			// origin, err = newNilRepoClone()
			// if err != nil {
			// 	return errors.E(op, p.UniquePath, err)
			// }
		}
		defer os.RemoveAll(originDir)

		// case kptfilev1.OciOrigin:
		// 	options := &[]crane.Option{crane.WithAuthFromKeychain(gcrane.Keychain)}
		// 	updatedDir, err = ioutil.TempDir("", "kpt-get-")
		// 	if err != nil {
		// 		return errors.E(op, errors.Internal, fmt.Errorf("error creating temp directory: %w", err))
		// 	}
		// 	defer os.RemoveAll(updatedDir)

		// 	if err = fetch.ClonerUsingOciPull(ctx, kf.Upstream.Oci.Image, &updatedDigest, updatedDir, options); err != nil {
		// 		return errors.E(op, p.UniquePath, err)
		// 	}

		// 	if kf.UpstreamLock != nil {
		// 		originDir, err = ioutil.TempDir("", "kpt-get-")
		// 		if err != nil {
		// 			return errors.E(op, errors.Internal, fmt.Errorf("error creating temp directory: %w", err))
		// 		}
		// 		defer os.RemoveAll(originDir)

		// 		if err = fetch.ClonerUsingOciPull(ctx, kf.UpstreamLock.Oci.Image, nil, originDir, options); err != nil {
		// 			return errors.E(op, p.UniquePath, err)
		// 		}
		// 	} else {
		// 		origin, err := newNilRepoClone()
		// 		if err != nil {
		// 			return errors.E(op, p.UniquePath, err)
		// 		}
		// 		originDir = origin.AbsPath()
		// 		defer os.RemoveAll(originDir)
		// 	}
	}

	// s := stack.New()
	// s.Push(".")

	// for s.Len() > 0 {
	{
		// relPath := s.Pop()
		relPath := "."
		localPath := filepath.Join(packageDir, relPath)
		updatedPath := filepath.Join(updated.AbsPath(), relPath)
		originPath := filepath.Join(originDir, relPath)
		isRoot := false
		if relPath == "." {
			isRoot = true
		}

		// if err := u.updatePackage(ctx, relPath, localPath, updatedPath, originPath, isRoot); err != nil {
		// 	return errors.E(op, p.UniquePath, err)
		// }

		updateOptions := kptlib.UpdateOptions{
			RelPackagePath: relPath,
			LocalPath:      localPath,
			UpdatedPath:    updatedPath,
			OriginPath:     originPath,
			IsRoot:         isRoot,
		}
		if err := kptlib.UpdateResourceMerge(ctx, updateOptions); err != nil {
			return err
		}

		// paths, err := pkgutil.FindSubpackagesForPaths(pkg.Remote, false,
		// 	localPath, updatedPath, originPath)
		// if err != nil {
		// 	return errors.E(op, p.UniquePath, err)
		// }
		// for _, path := range paths {
		// 	s.Push(filepath.Join(relPath, path))
		// }
	}

	updatedRepoSpec := updated.GitRepoSpec()
	if err := kptlib.UpdateUpstreamLockFromGit(packageDir, &updatedRepoSpec); err != nil {
		return err // errors.E(op, p.UniquePath, err)
	}

	return nil
}
