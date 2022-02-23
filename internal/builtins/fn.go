package builtins

import (
	"io"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const pkgContextFile = "package-context.yaml"
const pkgContextName = "kptfile.kpt.dev"

type PackageContextGenerator struct {
}

func (pc *PackageContextGenerator) Run(r io.Reader, w io.Writer) error {
	rw := &kio.ByteReadWriter{
		Reader:                r,
		Writer:                w,
		KeepReaderAnnotations: true,
	}
	return framework.Execute(pc, rw)
}

func (pc *PackageContextGenerator) Process(resourceList *framework.ResourceList) error {
	pkgContextAlreadyExists := false
	for _, resource := range resourceList.Items {
		if resource.GetName() == pkgContextName &&
			resource.GetApiVersion() == "v1" &&
			resource.GetKind() == "ConfigMap" {
			pkgContextAlreadyExists = true
			break
		}
	}

	if !pkgContextAlreadyExists {
		kf := getKptFile(resourceList)
		if kf == nil {
			// Strange: kptfile is missing
			// may be function was run without --include-meta-resources
			return nil
		}
		pkgContext, err := pkgContextNode(kf)
		if err != nil {
			resourceList.Results = framework.Results{
				&framework.Result{
					Message:  err.Error(),
					Severity: framework.Error,
				},
			}
			return resourceList.Results
		}
		resourceList.Items = append(resourceList.Items, pkgContext)
	}
	// Notify users the gcloud context is stored in `gcloud-config.yaml`.
	resourceList.Results = append(resourceList.Results, &framework.Result{
		Message:  "generated package context",
		Severity: framework.Info,
		File:     &framework.File{Path: pkgContextFile, Index: 0},
	})
	return nil
}

func getKptFile(rl *framework.ResourceList) *yaml.RNode {
	for _, resource := range rl.Items {
		gvk := resid.GvkFromNode(resource)
		if gvk.Kind == "Kptfile" && gvk.ApiVersion() == "kpt.dev/v1" {
			// return the first Kptfile resource
			// TODO(droot): Add support for nested packages
			return resource
		}
	}
	return nil
}

// NewGcloudConfigNode creates a `GcloudConfig` RNode resource.
func pkgContextNode(kf *yaml.RNode) (*yaml.RNode, error) {
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
	annotations := map[string]string{
		filters.LocalConfigAnnotation: "true",
		kioutil.PathAnnotation:        pkgContextFile,
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
