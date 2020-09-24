// Copyright 2019 Google LLC
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

package manifestreader

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/pathutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// KyamlPathManifestReader provides functionality for reading manifests
// and returning them as infos objects. It will ignore resources with the
// config.kubernetes.io/local-config annotation and set the namespace for
// namespace-scoped resources without a namespace.
type KyamlPathManifestReader struct {
	Path string

	manifestreader.ReaderOptions
}

// Read reads the manifests and returns them as Info objects.
func (p *KyamlPathManifestReader) Read() ([]*resource.Info, error) {
	var infos []*resource.Info
	paths, err := pathutil.DirsWithFile(p.Path, kptfile.KptFileName, true)
	if err != nil {
		return infos, err
	}

	_, err = os.Stat(filepath.Join(p.Path, kptfile.KptFileName))
	if err != nil {
		return infos, err
	}

	for _, p := range paths {
		nodes, err := (&kio.LocalPackageReader{
			PackagePath:     p,
			PackageFileName: kptfile.KptFileName,
		}).Read()
		if err != nil {
			return infos, err
		}

		for _, n := range nodes {
			fileName, err := extractFileName(n)
			if err != nil {
				return infos, err
			}

			err = removeAnnotations(n)
			if err != nil {
				return infos, err
			}
			inf, err := kyamlNodeToInfo(n, fileName)
			if err != nil {
				return infos, err
			}
			infos = append(infos, inf)
		}
	}

	infos = manifestreader.FilterLocalConfig(infos)

	err = manifestreader.SetNamespaces(p.Factory, infos, p.Namespace, p.EnforceNamespace)
	return infos, err
}

// extractFileName looks up the fileName from which the resource was read
// by lookup up the Path annotation set by the LocalPackageReader.
func extractFileName(n *yaml.RNode) (string, error) {
	val, err := n.Pipe(yaml.GetAnnotation(kioutil.PathAnnotation))
	if err != nil {
		return "", err
	}
	return val.YNode().Value, nil
}

var annotations = []kioutil.AnnotationKey{
	kioutil.PathAnnotation,
	kioutil.IndexAnnotation,
}

// removeAnnotations removes any Path or Index annotations from the
// resource.
func removeAnnotations(n *yaml.RNode) error {
	for _, a := range annotations {
		err := n.PipeE(yaml.ClearAnnotation(a))
		if err != nil {
			return err
		}
	}
	return nil
}

// kyamlNodeToInfo take a resource represented as a kyaml RNode and
// turns it into an info object with the underlying object represented
// as an unstructured. The provided fileName is used to set the source
// of the info.
func kyamlNodeToInfo(n *yaml.RNode, fileName string) (*resource.Info, error) {
	meta, err := n.GetMeta()
	if err != nil {
		return nil, err
	}

	b, err := n.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	return &resource.Info{
		Object: &unstructured.Unstructured{
			Object: m,
		},
		Name:      meta.Name,
		Namespace: meta.Namespace,
		Source:    fileName,
	}, nil
}
