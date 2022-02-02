// Copyright 2021 Google LLC
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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/pkg/content/open"
	"github.com/GoogleContainerTools/kpt/pkg/location"
)

func main() {
	run("temp folder examples", tempFolderExamples)

}

func run(caption string, example func() error) {
	if err := example(); err != nil {
		fmt.Printf("error in %q: %v\n", caption, err)
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

func tempFolderExamples() error {
	// set up temp dir
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	if err := os.WriteFile(filepath.Join(tmp, "hello.txt"), []byte("world"), os.ModePerm); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, "hello.yaml"), []byte(helloYaml), os.ModePerm); err != nil {
		return err
	}

	return runAll(
		location.Dir{
			Directory: tmp,
		},
	)
}

func runAll(ref location.Reference) error {
	if err := openContent(ref); err != nil {
		return fmt.Errorf("open content: %v", err)
	}
	if err := readFromFS(ref); err != nil {
		return fmt.Errorf("open content as FS: %v", err)
	}
	if err := readFromFileSystem(ref); err != nil {
		return fmt.Errorf("open content as FileSystem: %v", err)
	}
	if err := readFromReader(ref); err != nil {
		return fmt.Errorf("open content as Reader: %v", err)
	}
	return nil
}

func openContent(ref location.Reference) error {
	// open it as content
	src, err := open.Content(ref)
	if err != nil {
		return err
	}
	defer src.Close()

	fmt.Printf("opened %v\n", src.Location)
	return nil
}

func readFromFS(ref location.Reference) error {
	// open it as content as well as FS
	src, err := open.FS(ref)
	if err != nil {
		return err
	}
	defer src.Close()

	b, err := fs.ReadFile(src.FS, "hello.txt")
	if err != nil {
		return err
	}
	fmt.Printf("read %q\n", b)
	return nil
}

func readFromFileSystem(ref location.Reference) error {
	// open it as content as well as FS
	src, err := open.FileSystem(ref)
	if err != nil {
		return err
	}
	defer src.Close()

	b, err := src.FileSystem.ReadFile(filepath.Join(src.Path, "hello.txt"))
	if err != nil {
		return err
	}
	fmt.Printf("read %q\n", b)
	return nil
}

func readFromReader(ref location.Reference) error {
	// open it as content as well as FS
	src, err := open.Reader(ref)
	if err != nil {
		return err
	}
	defer src.Close()

	nodes, err := src.Reader.Read()
	if err != nil {
		return err
	}
	for _, node := range nodes {
		s, _ := node.String()
		fmt.Printf("read %v\n", s)
	}
	return nil
}
