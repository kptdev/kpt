package cmdrender

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"testing/fstest"

	"github.com/mholt/archiver/v4"
)

var deployment = `# deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: in-memory-nginx-deployment
spec:
  replicas: 3
`

var kptFile = `# Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: app
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-namespace:v0.1.3
      configMap:
        namespace: staging
`

func inMemPkg() fs.FS {

	rootFS := fstest.MapFS{
		"deployment.yaml": {Data: []byte(deployment)},
		"Kptfile":         {Data: []byte(kptFile)},
	}

	err := fs.WalkDir(rootFS, ".", func(path string, d fs.DirEntry, err error) error {
		fmt.Println(path)
		return nil
	})
	if err != nil {
		panic(err)
	}

	return rootFS
}

func openArchive(filename string) (fs.FS, error) {
	fsys, err := archiver.FileSystem(filename)
	if err != nil {
		return nil, err
	}

	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fmt.Println("Walking:", path, "Dir?", d.IsDir())
		if path == "." {
			// root directory
			return nil
		}
		if filepath.Base(path) == ".git" {
			fmt.Println("Skipping:", path)
			return fs.SkipDir
		}
		return nil
	})
	return fsys, nil
}
