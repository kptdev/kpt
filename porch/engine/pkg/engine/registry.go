// Copyright 2022 Google LLC
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

package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

type CaDRegistry interface {
	// FunctionImageForGVK returns the default function image for the provided function-config GVK
	FunctionImageForGVK(ctx context.Context, gvk schema.GroupVersionKind) (string, error)
}

type ociCaDRegistry struct {
	crdRegistry string
}

var _ CaDRegistry = &ociCaDRegistry{}

func NewOCIRegistry(registry string) (*ociCaDRegistry, error) {
	if !strings.HasSuffix(registry, "/") {
		registry += "/"
	}
	crdRegistry := registry + "crds/"

	return &ociCaDRegistry{
		crdRegistry: crdRegistry,
	}, nil
}

func (r *ociCaDRegistry) FunctionImageForGVK(ctx context.Context, gvk schema.GroupVersionKind) (string, error) {
	annotations, err := r.findCRDAnnotations(ctx, gvk)
	if err != nil {
		return "", fmt.Errorf("error getting CRD annotations for %v: %w", gvk, err)
	}

	functionImage := annotations["kpt.dev/function"]
	if functionImage != "" {
		return functionImage, nil
	}

	return "", fmt.Errorf("unable to determine function for GVK %v", gvk)
}

func (r *ociCaDRegistry) referenceForCRD(gvk schema.GroupVersionKind) (name.Reference, error) {
	// We can't have upper-case values
	// TODO: convert to e.g. snake case?
	kindPath := strings.ToLower(gvk.Kind)

	// TODO: Split group and reverse?
	crdImage := r.crdRegistry + gvk.Group + "/" + kindPath + ":" + gvk.Version

	// TODO: Support multiple versions?
	// tags, err := remote.List(crdImageRef, remoteOptions...)
	// if err != nil {
	// 	return nil, fmt.Errorf("error listing tags for %v: %w", crdImage, err)
	// }
	//
	// var candidates []string
	// for _, tag := range tags {
	// 	if tag == gvk.Version || strings.HasPrefix(tag, gvk.Version+"-") {
	// 		candidates = tag
	// 	}
	// }

	crdImageRef, err := name.ParseReference(crdImage)
	if err != nil {
		return nil, fmt.Errorf("error parsing image name %q: %w", crdImage, err)
	}
	return crdImageRef, nil
}

func (r *ociCaDRegistry) findCRDAnnotations(ctx context.Context, gvk schema.GroupVersionKind) (map[string]string, error) {
	ctx, span := tracer.Start(ctx, "ociCaDRegistry::findCRDAnnotations", trace.WithAttributes(
		attribute.String("group", gvk.Group),
		attribute.String("version", gvk.Version),
		attribute.String("kind", gvk.Kind),
	))
	defer span.End()

	var remoteOptions []remote.Option
	remoteOptions = append(remoteOptions, remote.WithAuthFromKeychain(gcrane.Keychain))
	remoteOptions = append(remoteOptions, remote.WithContext(ctx))

	crdImageRef, err := r.referenceForCRD(gvk)
	if err != nil {
		return nil, fmt.Errorf("error constructing name for CRD %v: %w", gvk, err)
	}

	img, err := remote.Image(crdImageRef, remoteOptions...)
	if err != nil {
		return nil, fmt.Errorf("error querying image %v: %w", crdImageRef, err)
	}
	// TODO: Cache based on digest
	// digest, err := img.Digest()
	manifest, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("error getting image manifest for %v: %w", crdImageRef, err)
	}
	klog.Infof("image %s has annotations %v", crdImageRef, manifest.Annotations)

	return manifest.Annotations, nil
}
