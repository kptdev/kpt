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

package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// Server is a mock OCI server implementing "just enough" of the oci protocol
type Server struct {
	registries Registries
	endpoint   string
}

// ServerOption follows the option pattern for customizing the server
type ServerOption interface {
	apply(*Server) error
}

// NewGitServer constructs a GitServer backed by the specified repo.
func NewServer(registries Registries, opts ...ServerOption) (*Server, error) {
	s := &Server{
		registries: registries,
	}

	for _, opt := range opts {
		if err := opt.apply(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// ListenAndServe starts the git server on "listen".
// The address we actually start listening on will be posted to addressChannel
func (s *Server) ListenAndServe(ctx context.Context, listen string, addressChannel chan<- net.Addr) error {
	httpServer := &http.Server{
		Addr:           listen,
		Handler:        s,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		close(addressChannel)
		return err
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctxWithCancel.Done()
		if err := httpServer.Shutdown(context.Background()); err != nil {
			klog.Warningf("error from oci httpServer.Shutdown: %v", err)
		}
		if err := httpServer.Close(); err != nil {
			klog.Warningf("error from oci httpServer.Close: %v", err)
		}
	}()

	s.endpoint = ln.Addr().String()

	addressChannel <- ln.Addr()

	return httpServer.Serve(ln)
}

func (s *Server) Endpoint() string {
	return s.endpoint
}

// ServeHTTP is the entrypoint for http requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := klog.FromContext(ctx)

	log.V(2).Info("http request", "method", r.Method, "url", r.URL)

	response, err := s.serveRequest(w, r)
	if err != nil {
		klog.Warningf("internal error from %s %s: %v", r.Method, r.URL, err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	response.WriteTo(w, r)
}

// serveRequest is the main dispatcher for http requests.
func (s *Server) serveRequest(w http.ResponseWriter, r *http.Request) (Response, error) {
	pathTokens := strings.Split(strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/"), "/"), "/")
	n := len(pathTokens)

	if n == 1 && pathTokens[0] == "v2" {
		return s.serveV2(w, r)
	}

	if len(pathTokens) >= 4 && pathTokens[0] == "v2" && pathTokens[n-2] == "tags" && pathTokens[n-1] == "list" {
		return s.serveTagsList(w, r, strings.Join(pathTokens[1:n-2], "/"))
	}

	if len(pathTokens) >= 4 && pathTokens[0] == "v2" && pathTokens[n-2] == "blobs" && strings.HasPrefix(pathTokens[n-1], "sha256:") {
		return s.serveBlob(w, r, strings.Join(pathTokens[1:n-2], "/"), pathTokens[n-1])
	}

	if len(pathTokens) >= 4 && pathTokens[0] == "v2" && pathTokens[n-2] == "blobs" && pathTokens[n-1] == "uploads" {
		return s.serveUploads(w, r, strings.Join(pathTokens[1:n-2], "/"))
	}

	if len(pathTokens) >= 5 && pathTokens[0] == "v2" && pathTokens[n-3] == "blobs" && pathTokens[n-2] == "uploads" {
		return s.serveUpload(w, r, strings.Join(pathTokens[1:n-3], "/"), pathTokens[n-1])
	}

	if len(pathTokens) >= 4 && pathTokens[0] == "v2" && pathTokens[n-2] == "manifests" {
		return s.serveManifest(w, r, strings.Join(pathTokens[1:n-2], "/"), pathTokens[n-1])
	}

	klog.Warningf("404 for %s %s", r.Method, r.URL)
	return ErrorResponse(http.StatusNotFound), nil
}

func (s *Server) serveV2(w http.ResponseWriter, r *http.Request) (Response, error) {
	if r.Method != "GET" && r.Method != "HEAD" {
		return ErrorResponse(http.StatusMethodNotAllowed), nil
	}
	return &TextResponse{}, nil
}

func (s *Server) serveTagsList(w http.ResponseWriter, r *http.Request, name string) (Response, error) {
	ctx := r.Context()

	if r.Method != "GET" && r.Method != "HEAD" {
		return ErrorResponse(http.StatusMethodNotAllowed), nil
	}

	repo, err := s.registries.FindRegistry(r.Context(), name)
	if err != nil {
		klog.Warningf("500 for %s %s: %v", r.Method, r.URL, err)
		return nil, err
	}

	tags, err := repo.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	return &JSONResponse{Object: tags}, nil
}

func (s *Server) openRegistry(r *http.Request, name string) (*Registry, Response, error) {
	repo, err := s.registries.FindRegistry(r.Context(), name)
	if err != nil {
		klog.Warningf("500 for %s %s: %v", r.Method, r.URL, err)
		return nil, ErrorResponse(http.StatusInternalServerError), nil
	}

	if repo == nil {
		// TODO: Should we send something consistent with auth failure?
		klog.Warningf("404 for %s %s (repo not found)", r.Method, r.URL)
		return nil, ErrorResponse(http.StatusNotFound), nil
	}

	if repo.username != "" || repo.password != "" {
		username, password, ok := r.BasicAuth()
		if !ok || username != repo.username || password != repo.password {
			return nil, ErrorResponse(http.StatusForbidden), nil
		}
	}

	return repo, nil, nil
}

func (s *Server) serveUploads(w http.ResponseWriter, r *http.Request, name string) (Response, error) {
	ctx := r.Context()

	if r.Method != "POST" {
		return ErrorResponse(http.StatusMethodNotAllowed), nil
	}

	repo, response, err := s.openRegistry(r, name)
	if response != nil || err != nil {
		return response, err
	}

	digest := r.PostFormValue("digest")
	if digest != "" {
		// TODO: single-post uploads
		return nil, fmt.Errorf("digest not implemented on /uploads/")
	}

	uuid, err := repo.StartUpload(ctx)
	if err != nil {
		return nil, err
	}

	return &HTTPResponse{
		Status:   http.StatusAccepted,
		Location: "/v2/" + name + "/blobs/uploads/" + uuid,
	}, nil
}

func (s *Server) serveUpload(w http.ResponseWriter, r *http.Request, name string, uuid string) (Response, error) {
	ctx := r.Context()

	switch r.Method {
	case "PUT", "PATCH":
		// ok
	default:
		return ErrorResponse(http.StatusMethodNotAllowed), nil
	}

	repo, response, err := s.openRegistry(r, name)
	if response != nil || err != nil {
		return response, err
	}

	contentLengthString := r.Header.Get("Content-Length")

	from := int64(0)
	to := int64(-1)

	pos, err := repo.UploadPosition(ctx, uuid)
	if err != nil {
		return nil, err
	}

	rangeNotSatisfiable := func() (*HTTPResponse, error) {
		return &HTTPResponse{
			Status:   http.StatusRequestedRangeNotSatisfiable,
			Location: "/v2/" + name + "/blobs/uploads/" + uuid,
			Range:    fmt.Sprintf("0-%d", pos),
		}, nil
	}
	if r.Method == "PATCH" || contentLengthString != "0" {
		if pos != from {
			klog.Warningf("write to wrong position for PATCH %v; pos=%d, from=%d", r.URL, pos, from)
			return rangeNotSatisfiable()
		}
	}

	contentRangeString := r.Header.Get("Content-Range")
	if contentRangeString != "" {
		tokens := strings.Split(contentRangeString, "-")
		if len(tokens) != 2 {
			klog.Warningf("invalid content-range %q for PATCH %v", contentRangeString, r.URL)
			return rangeNotSatisfiable()
		}

		from, err = strconv.ParseInt(tokens[0], 10, 64)
		if err != nil || from < 0 {
			klog.Warningf("invalid content-range %q for PATCH %v", contentRangeString, r.URL)
			return rangeNotSatisfiable()
		}

		to, err = strconv.ParseInt(tokens[1], 10, 64)
		if err != nil || to < 0 {
			klog.Warningf("invalid content-range %q for PATCH %v", contentRangeString, r.URL)
			return rangeNotSatisfiable()
		}
	}

	if r.Method == "PATCH" || contentLengthString != "0" {
		n, err := repo.AppendUpload(ctx, uuid, from, r.Body)
		if err != nil {
			return nil, fmt.Errorf("error writing content to upload file: %w", err)
		}
		if to >= 0 && from+n != to {
			klog.Warningf("received short content for PATCH %v: from=%d, to=%d, n=%d", r.URL, from, to, n)
			return ErrorResponse(http.StatusBadRequest), nil
		}
		to = from + n
	}

	if r.Method == "PUT" {
		digest := r.FormValue("digest")
		if digest == "" {
			klog.Warningf("digest is required for %v %v", r.Method, r.URL)
			return ErrorResponse(http.StatusBadRequest), nil
		}

		blobID, err := repo.CompleteUpload(ctx, uuid, digest)
		if err != nil {
			return nil, err
		}
		return &HTTPResponse{
			Status:   http.StatusCreated,
			Location: "/v2/" + name + "/blobs/" + blobID,
		}, nil
	}

	return &HTTPResponse{
		Status:   http.StatusAccepted,
		Location: "/v2/" + name + "/blobs/uploads/" + uuid,
		Range:    fmt.Sprintf("0-%d", to),
	}, nil
}

func (s *Server) serveBlob(w http.ResponseWriter, r *http.Request, name string, blob string) (Response, error) {
	ctx := r.Context()

	switch r.Method {
	case "GET", "HEAD":
		// ok
	default:
		return ErrorResponse(http.StatusMethodNotAllowed), nil
	}

	repo, response, err := s.openRegistry(r, name)
	if response != nil || err != nil {
		return response, err
	}

	return repo.ServeBlob(ctx, blob)
}

func (s *Server) serveManifest(w http.ResponseWriter, r *http.Request, name string, tag string) (Response, error) {
	ctx := r.Context()
	log := klog.FromContext(ctx)

	switch r.Method {
	case "PUT", "GET":
		// ok
	default:
		return ErrorResponse(http.StatusMethodNotAllowed), nil
	}

	repo, response, err := s.openRegistry(r, name)
	if response != nil || err != nil {
		return response, err
	}

	// TODO: Verify tag is "legal"

	switch r.Method {
	case "PUT":
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}

		// TODO: Verify name and tag match provided values
		// TODO: Verify layers are known to registry

		if err := repo.CreateManifest(ctx, tag, b); err != nil {
			return nil, err
		}

		return &HTTPResponse{
			Status: http.StatusCreated,
		}, nil

	case "GET":
		// We read the file because it's (typically) pretty small, and this is an easy way to get the correct content-type
		// Otherwise crane warns about the lack of a content-type
		b, err := repo.ReadManifest(ctx, tag)
		if err != nil {
			if os.IsNotExist(err) {
				log.Info("manifest not found", "manifest", tag)
				return ErrorResponse(http.StatusNotFound), nil
			}
			return nil, err
		}
		type manifestInfo struct {
			MediaType string `json:"mediaType"`
		}
		var manifest manifestInfo
		if err := json.Unmarshal(b, &manifest); err != nil {
			klog.Warningf("error unmarshaling manifest %q: %v", tag, err)
		}
		response := &BinaryResponse{
			Body: b,
		}
		if manifest.MediaType != "" {
			switch manifest.MediaType {
			case "application/vnd.docker.distribution.manifest.v2+json":
				response.ContentType = "application/vnd.docker.distribution.manifest.v2+json"
			case "application/vnd.oci.image.manifest.v1+json":
				response.ContentType = "application/vnd.oci.image.manifest.v1+json"
			}
		}
		if response.ContentType == "" {
			klog.Warningf("cannot determine media type for %q: %v", tag, string(b))
		}
		return response, nil

	default:
		return nil, fmt.Errorf("unexpected method")
	}
}
