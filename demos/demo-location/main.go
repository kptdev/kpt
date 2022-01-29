package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/GoogleContainerTools/kpt/pkg/location"
	"github.com/GoogleContainerTools/kpt/pkg/location/mutate"
)

func main() {
	ctx := context.Background()

	opts := []location.Option{
		location.WithContext(ctx),
	}

	fmt.Println("- parsing argument to location")
	fmt.Println()

	example(
		"oci example",
		"oci://us-docker.pkg.dev/my-project-id/my-repo-name/my-blueprint:draft",
		"example",
		"sha256:9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba",
		opts...,
	)

	example(
		"git example",
		"https://github.com/GoogleCloudPlatform/blueprints.git/catalog/gke@gke-blueprint-v0.4.0",
		"main",
		"2b8afca2ef0662cf5ea39c797832ac9c5ea67c7e",
		opts...,
	)

	example(
		"dir example",
		"path/to/dir",
		"qa",
		"",
		opts...,
	)

	// stdin and stdout options are added on individual calls to parse, because
	// only the caller knows if "-" in an argument means read from stdin or write to stdout
	example(
		"stdin example",
		"-",
		"",
		"",
		append(opts, location.WithStdin(os.Stdin))...,
	)

	example(
		"stdout example",
		"-",
		"",
		"",
		append(opts, location.WithStdin(os.Stdin))...,
	)

	example(
		"duplex example",
		"-",
		"",
		"",
		append(opts, location.WithStdin(os.Stdin), location.WithStdout(os.Stdout))...,
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

	updated, _ := mutate.Identifier(ref, "preview")
	fmt.Printf("updated: %v\n", updated)

	locked, _ := mutate.Lock(updated, "98510723450981325098375013")
	fmt.Printf("locked: %v\n", locked)
	fmt.Println()

}

func example(caption string, arg string, identifier string, hash string, opts ...location.Option) {
	fmt.Printf("%s %q\n", caption, arg)
	if err := run(arg, identifier, hash, opts...); err != nil {
		fmt.Printf("example error: %v\n", err)
	}
	fmt.Println()
}

func run(arg string, identifier string, hash string, opts ...location.Option) error {
	parsed, err := location.ParseReference(arg, opts...)
	if err != nil {
		return err
	}
	fmt.Printf("parsed: {%v}\n", parsed)

	if identifier != "" {
		changed, err := mutate.Identifier(parsed, identifier)
		if err != nil {
			return err
		}
		fmt.Printf("changed: {%v}\n", changed)

		if hash != "" {
			locked, err := mutate.Lock(changed, hash)
			if err != nil {
				return err
			}
			fmt.Printf("locked: {%v}\n", locked)
		}
	}

	return nil
}

// example of providing a custom location that supports branch/tag identifier and locking to a unique id.
// the meaning of the fields in the struct are entirely specific to the location type.

type CustomLocation struct {
	WhereItIs            string
	LabelOrVersionString string
}

type CustomLocationLock struct {
	WhereItIs            string
	LabelOrVersionString string
	UniqueIdString       string
}

// compile-time check that duck types are correct
var _ location.Reference = CustomLocation{}
var _ location.ReferenceLock = CustomLocationLock{}
var _ mutate.IdentifierSetter = CustomLocation{}
var _ mutate.LockSetter = CustomLocation{}

// string when reference appears in console and log messages
func (ref CustomLocation) String() string {
	return fmt.Sprint(" WhereItIs:", ref.WhereItIs, " LabelOrVersionString:", ref.LabelOrVersionString)
}

// string when reference appears in console and log messages
func (ref CustomLocationLock) String() string {
	return fmt.Sprint(" WhereItIs:", ref.WhereItIs, " LabelOrVersionString:", ref.LabelOrVersionString, " UniqueIdString:", ref.UniqueIdString)
}

// return location with only the identifier changed
// depending on location, the exact meaning may be branch/label/version/tag/etc
func (ref CustomLocation) SetIdentifier(name string) (location.Reference, error) {
	return CustomLocation{
		WhereItIs:            ref.WhereItIs,
		LabelOrVersionString: name,
	}, nil
}

// return the locked form of the location
// depending on location, the exact meaning may be commit-id, image-digest, url-query parameter, exact resource name, etc.
func (ref CustomLocation) SetLock(uniqueId string) (location.ReferenceLock, error) {
	return CustomLocationLock{
		WhereItIs:            ref.WhereItIs,
		LabelOrVersionString: ref.LabelOrVersionString,
		UniqueIdString:       uniqueId,
	}, nil
}
