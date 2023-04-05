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

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"k8s.io/klog/v2"
)

func main() {
	err := run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}

func run(ctx context.Context) error {
	listen := ":8081"
	flag.StringVar(&listen, "listen", listen, "endpoint on which to server HTTP")
	baseDir := "../../site"
	flag.StringVar(&baseDir, "base", baseDir, "directory containing static content")
	flag.Parse()

	s := &Server{files: make(map[string]*staticContent)}

	if err := s.AddDirectory(baseDir, "/"); err != nil {
		return err
	}
	return s.ListenAndServe(listen)
}

type Server struct {
	files map[string]*staticContent
}

type staticContent struct {
	data []byte
}

// AddDirectory will load all the files from the directory tree `dir` and register them to be served under `urlPath`
func (s *Server) AddDirectory(dir string, urlPath string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error from ReadDir(%q): %w", dir, err)
	}
	for _, file := range files {
		if file.IsDir() {
			p := filepath.Join(dir, file.Name())
			u, err := urlJoinPath(urlPath, file.Name())
			if err != nil {
				return err
			}

			if err := s.AddDirectory(p, u); err != nil {
				return err
			}
		} else {
			if err := s.addFile(dir, urlPath, file); err != nil {
				return err
			}
		}
	}

	return nil
}

// urlJoinPath joins two urls, and should be replaced by url.JoinPath asap
func urlJoinPath(base string, ext string) (string, error) {
	if strings.HasSuffix(base, "/") {
		return base + ext, nil
	}
	return base + "/" + ext, nil
}

// addFile adds a single file to the content to be served.
// It currently understands .md as markdown files, and will autoconvert them to html.
func (s *Server) addFile(dir string, urlPath string, file fs.DirEntry) error {
	p := filepath.Join(dir, file.Name())

	b, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("failed to ReadFile(%q): %w", p, err)
	}

	urlName := file.Name()
	if filepath.Ext(urlName) == ".md" {
		md := goldmark.New()
		var buf bytes.Buffer
		if err := md.Convert(b, &buf); err != nil {
			return fmt.Errorf("error converting file %q from markdown: %v", p, err)
		}
		b = buf.Bytes()

		urlName = strings.TrimSuffix(urlName, ".md")
	}
	u, err := urlJoinPath(urlPath, urlName)
	if err != nil {
		return err
	}
	klog.Infof("registered content %q", u)
	s.files[u] = &staticContent{
		data: b,
	}

	return nil
}

// ListenAndServe starts an http.Server on addr
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s)
}

// ServeHTTP is the entrypoint for serving http requests
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	klog.Infof("request %s %s", r.Method, r.URL.Path)
	content := s.files[r.URL.Path]
	if content == nil {
		klog.Warningf("404 for %s", r.URL.Path)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Write(content.data)
}
