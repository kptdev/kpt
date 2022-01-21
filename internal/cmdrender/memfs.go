package cmdrender

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/mholt/archiver/v4"
	"github.com/psanford/memfs"
)

var deployment = `# deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
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
    - image: gcr.io/kpt-fn/set-labels:v0.1.4
      configMap:
        tier: backend
`

func makeMemPkg() (fs.FS, string) {

	rootFS := memfs.New()

	/*
		err := rootFS.MkdirAll("a", 0777)
		if err != nil {
			panic(err)
		} */

	err := rootFS.WriteFile("deployment.yaml", []byte(deployment), 0755)
	if err != nil {
		panic(err)
	}

	err = rootFS.WriteFile("Kptfile", []byte(kptFile), 0755)
	if err != nil {
		panic(err)
	}

	err = fs.WalkDir(rootFS, ".", func(path string, d fs.DirEntry, err error) error {
		fmt.Println(path)
		return nil
	})
	if err != nil {
		panic(err)
	}

	return rootFS, "/"
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
