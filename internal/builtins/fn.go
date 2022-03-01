package builtins

import (
	"io"
	"path"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const pkgContextFile = "package-context.yaml"
const pkgContextName = "kptfile.kpt.dev"

// PackageContextGenerator is a built-in KRM function that generates
// a KRM object that contains package context information that can be
// used by functions such as `set-namespace` to customize package with
// minimal configuration.
type PackageContextGenerator struct{}

func (pc *PackageContextGenerator) Run(r io.Reader, w io.Writer) error {
	rw := &kio.ByteReadWriter{
		Reader:                r,
		Writer:                w,
		KeepReaderAnnotations: true,
	}
	return framework.Execute(pc, rw)
}

func (pc *PackageContextGenerator) Process(resourceList *framework.ResourceList) error {
	var err error
	pkgContexts := map[string]*yaml.RNode{}
	var updatedResources []*yaml.RNode

	for _, resource := range resourceList.Items {
		gvk := resid.GvkFromNode(resource)
		resourcePath, _, err := kioutil.GetFileAnnotations(resource)
		if err != nil {
			return err
		}
		if gvk.Kind == "Kptfile" && gvk.ApiVersion() == "kpt.dev/v1" {
			pkgContext, err := pkgContextResource(resource)
			if err != nil {
				return err
			}
			pkgContextFilepath, _, err := kioutil.GetFileAnnotations(pkgContext)
			if err != nil {
				return err
			}
			pkgContexts[pkgContextFilepath] = pkgContext
		}

		if gvk.Kind == "ConfigMap" &&
			gvk.ApiVersion() == "v1" &&
			strings.HasSuffix(resourcePath, pkgContextFile) {
			// skip adding pkg contexts
			continue
		}
		updatedResources = append(updatedResources, resource)
	}

	for _, resource := range pkgContexts {
		updatedResources = append(updatedResources, resource)
	}

	resourceList.Items = updatedResources

	if err != nil {
		resourceList.Results = framework.Results{
			&framework.Result{
				Message:  err.Error(),
				Severity: framework.Error,
			},
		}
		return resourceList.Results
	}
	// Notify users the gcloud context is stored in `gcloud-config.yaml`.
	resourceList.Results = append(resourceList.Results, &framework.Result{
		Message:  "generated package context",
		Severity: framework.Info,
		File:     &framework.File{Path: pkgContextFile, Index: 0},
	})
	return nil
}

// NewGcloudConfigNode creates a `GcloudConfig` RNode resource.
func pkgContextResource(kf *yaml.RNode) (*yaml.RNode, error) {
	cm := yaml.MustParse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name:
data: {}
`)
	// !! The ConfigMap should always be assigned to this value to make it "convention over configuration".
	if err := cm.SetName(pkgContextName); err != nil {
		return nil, err
	}
	kptfilePath, _, err := kioutil.GetFileAnnotations(kf)
	if err != nil {
		return nil, err
	}
	annotations := map[string]string{
		filters.LocalConfigAnnotation: "true",
		kioutil.PathAnnotation:        path.Join(path.Dir(kptfilePath), pkgContextFile),
	}
	// This resource is pseudo resource and not expected to be deployed to a cluster.
	if err := cm.SetAnnotations(annotations); err != nil {
		return nil, err
	}
	cm.SetDataMap(map[string]string{
		"name": kf.GetName(),
	})
	return cm, nil
}
