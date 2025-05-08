package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/klog/v2"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	listen := ":8080"
	staticRoot := "./kpt.dev"
	if s := os.Getenv("KO_DATA_PATH"); s != "" {
		staticRoot = filepath.Join(s, "static")
	}
	flag.Parse()

	httpRoot := http.Dir(staticRoot)
	httpFileServer := http.FileServer(httpRoot)

	allFiles := make(map[string]fs.DirEntry)
	if err := fs.WalkDir(os.DirFS(staticRoot), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		path = "/" + path
		allFiles[path] = d
		klog.Infof("path %v", path)
		return nil
	}); err != nil {
		return err
	}

	// rewrites contains a map of all the redirects we support, such as foo => foo.html
	rewrites := make(map[string]string)
	for k := range allFiles {
		if strings.HasSuffix(k, ".html") {
			// foo => foo.html
			rewrites[strings.TrimSuffix(k, ".html")] = k
			// foo/ => foo.html
			rewrites[strings.TrimSuffix(k, ".html")+"/"] = k
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
			r.URL.Path = upath
		}

		rewrite, ok := rewrites[r.URL.Path]
		if ok {
			r.URL.Path = rewrite
		}

		httpFileServer.ServeHTTP(w, r)
	})

	klog.Infof("serving %q on %v", staticRoot, listen)
	err := http.ListenAndServe(listen, nil)
	if err != nil {
		return fmt.Errorf("error serving on %q: %w", listen, err)
	}
	return nil
}
