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
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/GoogleContainerTools/kpt/pkg/location"
	"github.com/GoogleContainerTools/kpt/pkg/location/extensions"
	"github.com/GoogleContainerTools/kpt/pkg/location/mutate"
)

func main() {
	ctx := context.Background()

	opts := location.Options(
		location.WithContext(ctx),
		location.WithParsers(
			location.StdioParser,
			CustomParser,
			location.GitParser,
			location.OciParser,
			location.DirParser,
		),
	)

	fmt.Println("- parsing argument to location")
	fmt.Println()

	example(
		"oci example",
		"oci://us-docker.pkg.dev/my-project-id/my-repo-name/my-blueprint//nodepools/primary:draft",
		"example",
		"sha256:9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba",
		opts,
	)

	example(
		"git example",
		"https://github.com/GoogleCloudPlatform/blueprints.git/catalog/gke@gke-blueprint-v0.4.0",
		"main",
		"2b8afca2ef0662cf5ea39c797832ac9c5ea67c7e",
		opts,
	)

	example(
		"dir example",
		"path/to/dir",
		"qa",
		"",
		opts,
	)

	example(
		"custom example",
		"custom:where:which",
		"another",
		"98175",
		opts,
	)

	// stdin and stdout options are added on individual calls to parse, because
	// only the caller knows if "-" in an argument means read from stdin or write to stdout
	example(
		"stdin example",
		"-",
		"",
		"",
		location.WithStdin(os.Stdin),
		opts,
	)

	example(
		"stdout example",
		"-",
		"",
		"",
		location.WithStdout(os.Stdout),
		opts,
	)

	fmt.Println("- creating locations directly")
	fmt.Println()

	fmt.Println("stdio example")
	in := location.InputStream{
		Reader: os.Stdin,
	}
	out := location.OutputStream{
		Writer: os.Stdout,
	}
	fmt.Printf("stdio in %v\n", in)
	fmt.Printf("stdio out %v\n", out)
	fmt.Println()

	fmt.Println("bytes example")
	buf := bytes.NewBuffer([]byte("hello world"))
	in = location.InputStream{
		Reader: buf,
	}
	out = location.OutputStream{
		Writer: buf,
	}
	fmt.Printf("buffer in %v\n", in)
	fmt.Printf("buffer out %v\n", out)
	fmt.Println()

	fmt.Println("custom location example")

	ref := CustomLocation{
		WhereItIs:            "http://127.0.0.1:8001/my-package",
		LabelOrVersionString: "draft",
	}
	fmt.Printf("ref: %v\n", ref)

	updated, _ := location.SetRevision(ref, "preview")
	fmt.Printf("updated: %v\n", updated)

	locked, _ := mutate.Lock(updated, "98510723450981325098375013")
	fmt.Printf("locked: %v\n", locked)
	fmt.Println()
}

func example(caption string, arg string, revision string, lock string, opts ...location.Option) {
	fmt.Printf("%s %q\n", caption, arg)
	if err := run(arg, revision, lock, opts...); err != nil {
		fmt.Printf("example error: %v\n", err)
	}
	fmt.Println()
}

func run(arg string, revision string, lock string, opts ...location.Option) error {
	// parse arg to a reference
	parsed, err := location.ParseReference(arg, opts...)
	if err != nil {
		return err
	}
	fmt.Printf("parsed: %s\n", parsed)

	var changed location.Reference
	if revision != "" {
		// changing reference's pkg revision field
		changed, err = location.SetRevision(parsed, revision)
		if err != nil {
			return err
		}
		fmt.Printf("changed: %s\n", changed)
	} else {
		changed = parsed
	}

	if lock != "" {
		// making a locked reference with the unique value field for that reference type
		locked, err := mutate.Lock(changed, lock)
		if err != nil {
			return err
		}
		fmt.Printf("locked: %s\n", locked)
	}

	return nil
}

// example of providing a custom location that supports branch/tag revision and locking to a unique id.
// the meaning of the fields in the struct are entirely specific to the location type.

type CustomLocation struct {
	WhereItIs            string
	LabelOrVersionString string
}

type CustomLocationLock struct {
	CustomLocation
	UniqueIDString string
}

// compile-time check that duck types are correct
var _ location.Reference = CustomLocation{}
var _ location.ReferenceLock = CustomLocationLock{}
var _ extensions.Revisable = CustomLocation{}
var _ mutate.LockSetter = CustomLocation{}

var CustomParser = location.NewParser(
	[]string{
		"Custom locations format is 'custom:[WHERE]:[LABEL-OR-VERSION]'",
	},
	func(parse *location.Parse) {
		parts := strings.SplitN(parse.Value, ":", 3)
		if len(parts) == 3 && parts[0] == "custom" {
			parse.Result(CustomLocation{
				WhereItIs:            parts[1],
				LabelOrVersionString: parts[2],
			})
		}
	},
)

// string when reference appears in console and log messages
func (ref CustomLocation) String() string {
	return fmt.Sprintf("custom:%s:%s", ref.WhereItIs, ref.LabelOrVersionString)
}

func (ref CustomLocation) Type() string {
	return "custom"
}

func (ref CustomLocation) Validate() error {
	return nil
}

// string when reference appears in console and log messages
func (ref CustomLocationLock) String() string {
	return fmt.Sprintf("custom:%s:%s@%s", ref.WhereItIs, ref.LabelOrVersionString, ref.UniqueIDString)
}

// GetRevision returns the value that may be branch/label/version/tag/etc
func (ref CustomLocation) GetRevision() (string, bool) {
	return ref.LabelOrVersionString, true
}

// SetRevision returns location with only the revision changed depending on location,
// the exact meaning may be branch/label/version/tag/etc
func (ref CustomLocation) SetRevision(labelOrVersion string) (location.Reference, error) {
	return CustomLocation{
		WhereItIs:            ref.WhereItIs,
		LabelOrVersionString: labelOrVersion,
	}, nil
}

// return the locked form of the location
// depending on location, the exact meaning may be commit-id, image-digest, url-query parameter, exact resource name, etc.
func (ref CustomLocation) SetLock(uniqueID string) (location.ReferenceLock, error) {
	return CustomLocationLock{
		CustomLocation: CustomLocation{
			WhereItIs:            ref.WhereItIs,
			LabelOrVersionString: ref.LabelOrVersionString,
		},
		UniqueIDString: uniqueID,
	}, nil
}
