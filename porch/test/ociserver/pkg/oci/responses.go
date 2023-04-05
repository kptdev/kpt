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
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"k8s.io/klog/v2"
)

type JSONResponse struct {
	Object interface{}
}

func (v *JSONResponse) WriteTo(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(v.Object)
	if err != nil {
		klog.Warningf("error converting response to json: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	klog.Infof("writing json response %v", string(b))
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

type BinaryResponse struct {
	Body        []byte
	ContentType string
}

func (v *BinaryResponse) WriteTo(w http.ResponseWriter, r *http.Request) {
	if v.ContentType != "" {
		w.Header().Set("Content-Type", v.ContentType)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(v.Body)
}

type TextResponse struct {
	Body string
}

func (v *TextResponse) WriteTo(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(v.Body))
}

type FileResponse struct {
	Stat        os.FileInfo
	ContentType string
	Path        string
}

func (v *FileResponse) WriteTo(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(v.Path)
	if err != nil {
		klog.Warningf("error opening file %q: %v", v.Path, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if v.ContentType != "" {
		w.Header().Add("Content-Type", v.ContentType)
	}
	w.Header().Add("Content-Length", strconv.FormatInt(v.Stat.Size(), 10))
	w.WriteHeader(http.StatusOK)

	io.Copy(w, f)
}

type StreamingResponse struct {
	ContentType string
	Body        io.ReadCloser
}

func (v *StreamingResponse) WriteTo(w http.ResponseWriter, r *http.Request) {
	defer v.Body.Close()

	if v.ContentType != "" {
		w.Header().Add("Content-Type", v.ContentType)
	}
	// w.Header().Add("Content-Length", strconv.FormatInt(v.Stat.Size(), 10))
	w.WriteHeader(http.StatusOK)

	io.Copy(w, v.Body)
}

type HTTPResponse struct {
	Status      int
	Location    string
	ContentType string
	Range       string
}

func (v *HTTPResponse) WriteTo(w http.ResponseWriter, r *http.Request) {
	if v.Location != "" {
		w.Header().Set("Location", v.Location)
	}
	if v.Range != "" {
		w.Header().Set("Range", v.Range)
	}
	if v.ContentType != "" {
		w.Header().Set("Content-Type", v.ContentType)
	}
	http.Error(w, http.StatusText(v.Status), v.Status)
}

func ErrorResponse(statusCode int) *HTTPResponse {
	return &HTTPResponse{Status: statusCode}
}

type Response interface {
	WriteTo(w http.ResponseWriter, r *http.Request)
}
