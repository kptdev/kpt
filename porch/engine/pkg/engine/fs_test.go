package engine

import "testing"

func TestCreate(t *testing.T) {
	fs := &memfs{}

	fs.MkdirAll("a/b/c/")
	fs.MkdirAll("/d/e/f")
	fs.WriteFile("/a/b/c/foo.yaml", []byte("Hello World"))
}
