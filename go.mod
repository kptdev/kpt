module github.com/GoogleContainerTools/kpt

go 1.14

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.0
	github.com/evanphx/json-patch/v5 v5.2.0
	github.com/go-errors/errors v1.0.1
	github.com/igorsobreira/titlecase v0.0.0-20140109233139-4156b5b858ac
	github.com/philopon/go-toposort v0.0.0-20170620085441-9be86dbd762f
	github.com/posener/complete/v2 v2.0.1-alpha.12
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	gotest.tools v2.2.0+incompatible
	k8s.io/apiextensions-apiserver v0.18.10
	k8s.io/apimachinery v0.21.0
	k8s.io/cli-runtime v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/klog v1.0.0
	k8s.io/kubectl v0.21.0
	sigs.k8s.io/cli-utils v0.25.1-0.20210521231537-8200fe56434d
	sigs.k8s.io/kustomize/kyaml v0.10.20-0.20210506224302-fcfdf6be5152
	sigs.k8s.io/yaml v1.2.0
)

// TODO: sigs.k8s.io/cli-utils@v0.25.0 is still using old version of cli-runtime
// and kubectl and ultimately depends on old version of kustomize which will make
// license check fail
