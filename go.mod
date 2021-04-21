module github.com/GoogleContainerTools/kpt

go 1.14

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.0
	github.com/evanphx/json-patch/v5 v5.2.0 // indirect
	github.com/go-errors/errors v1.0.1
	github.com/go-openapi/spec v0.19.5
	github.com/igorsobreira/titlecase v0.0.0-20140109233139-4156b5b858ac
	// TODO: find a library that have proper releases or just implement
	// topsort in kpt.
	github.com/philopon/go-toposort v0.0.0-20170620085441-9be86dbd762f
	github.com/posener/complete/v2 v2.0.1-alpha.12
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/xlab/treeprint v1.1.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	gotest.tools v2.2.0+incompatible
	k8s.io/apimachinery v0.20.4
	k8s.io/cli-runtime v0.20.4
	k8s.io/client-go v0.20.4
	k8s.io/klog v1.0.0
	k8s.io/kubectl v0.20.4
	sigs.k8s.io/cli-utils v0.25.0
	sigs.k8s.io/kustomize/kyaml v0.10.17
	sigs.k8s.io/yaml v1.2.0 // indirect
)
