module github.com/GoogleContainerTools/kpt

go 1.14

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.0
	github.com/go-errors/errors v1.4.0
	github.com/go-openapi/spec v0.19.5
	github.com/olekukonko/tablewriter v0.0.4
	github.com/pkg/errors v0.9.1
	github.com/posener/complete/v2 v2.0.1-alpha.12
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/cli-runtime v0.21.1
	k8s.io/client-go v0.21.1
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20210421082810-95288971da7e
	k8s.io/kubectl v0.21.1
	sigs.k8s.io/cli-utils v0.25.1-0.20210603052138-670dee18a123
	sigs.k8s.io/kustomize/api v0.8.10 // indirect
	sigs.k8s.io/kustomize/cmd/config v0.9.12
	sigs.k8s.io/kustomize/kyaml v0.10.21
)

replace gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
