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

package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/pkg/content/open"
	"github.com/GoogleContainerTools/kpt/pkg/location"
)

func main() {
	// wire the global printer
	pr := printer.New(os.Stdout, os.Stderr)

	// create context with associated printer
	ctx := printer.WithContext(context.Background(), pr)

	// run through samples on location.Dir{}
	fmt.Print("\n## Temp folder example\n\n")
	if err := tempFolderExample(ctx); err != nil {
		fmt.Printf("error in tempFolderExamples: %v\n", err)
	}

	// run through examples on location.Git{}
	fmt.Print("\n## GitHub blueprint example\n\n")
	if err := githubBlueprintExample(ctx); err != nil {
		fmt.Printf("error in githubBlueprintExample: %v\n", err)
	}

	if len(os.Args) == 2 {
		// run through examples on provided location
		fmt.Print("\n## Parsed location example\n\n")
		if err := locationExample(ctx, os.Args[1]); err != nil {
			fmt.Printf("error in locationExample: %v\n", err)
		}
	}
}

var helloYaml = `
apiVersion: example.com/v1
kind: Greeting
metadata:
  name: hello
spec:
  target: world
`

func tempFolderExample(ctx context.Context) error {
	// set up temp dir
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte("Hello world"), os.ModePerm); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, "hello.yaml"), []byte(helloYaml), os.ModePerm); err != nil {
		return err
	}

	return runAll(
		ctx,
		location.Dir{
			Directory: tmp,
		},
	)
}

func githubBlueprintExample(ctx context.Context) error {
	return runAll(
		ctx,
		location.Git{
			Repo:      "https://github.com/GoogleCloudPlatform/blueprints",
			Directory: "catalog/gke",
			Ref:       "main",
		},
	)
}

func locationExample(ctx context.Context, arg string) error {
	ref, err := location.ParseReference(
		arg,
		location.WithContext(ctx),
		location.WithParsers(location.OciParser, location.GitParser, location.DirParser),
	)
	if err != nil {
		return err
	}

	return runAll(ctx, ref)
}

func runAll(ctx context.Context, ref location.Reference) error {
	if err := openContent(ctx, ref); err != nil {
		return fmt.Errorf("open content: %v", err)
	}
	if err := readFromFileSystem(ctx, ref); err != nil {
		return fmt.Errorf("open content as FileSystem: %v", err)
	}
	if err := readFromReader(ctx, ref); err != nil {
		return fmt.Errorf("open content as Reader: %v", err)
	}
	if err := readFromFS(ctx, ref); err != nil {
		return fmt.Errorf("open content as FS: %v", err)
	}
	return nil
}

func openContent(ctx context.Context, ref location.Reference) error {
	// open it as content
	src, err := open.Content(ref, open.WithContext(ctx))
	if err != nil {
		return err
	}
	defer src.Close()

	fmt.Printf("opened %v\n", src.Reference)
	return nil
}

func readFromFS(ctx context.Context, ref location.Reference) error {
	// open it as content as well as FS
	src, err := open.FS(ref, open.WithContext(ctx))
	if err != nil {
		return err
	}
	defer src.Close()

	b, err := fs.ReadFile(src.FS, "README.md")
	if err != nil {
		return err
	}
	fmt.Printf("read %q\n", short(b, 30))
	return nil
}

func readFromFileSystem(ctx context.Context, ref location.Reference) error {
	// open it as content as well as FS
	src, err := open.FileSystem(ref, open.WithContext(ctx))
	if err != nil {
		return err
	}
	defer src.Close()

	b, err := src.FileSystem.ReadFile(filepath.Join(src.Path, "README.md"))
	if err != nil {
		return err
	}
	fmt.Printf("read %q\n", short(b, 30))
	return nil
}

func short(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[0:n]
}

func readFromReader(ctx context.Context, ref location.Reference) error {
	// open it as content as well as FS
	src, err := open.Reader(ref, open.WithContext(ctx))
	if err != nil {
		return err
	}
	defer src.Close()

	nodes, err := src.Reader.Read()
	if err != nil {
		return err
	}
	for _, node := range nodes {
		m, _ := node.GetMeta()

		fmt.Printf("read %v\n", m.GetIdentifier())
	}
	return nil
}
