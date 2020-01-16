module github.com/GoogleContainerTools/kpt

go 1.13

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.0
	github.com/go-errors/errors v1.0.1
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/olekukonko/tablewriter v0.0.4
	github.com/posener/complete/v2 v2.0.1-alpha.12
	github.com/prometheus/client_golang v1.3.0 // indirect; https://github.com/golang/go/issues/34461
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	gopkg.in/yaml.v2 v2.2.7 // indirect
	gopkg.in/yaml.v3 v3.0.0-20191026110619-0b21df46bc1d
	gotest.tools v2.2.0+incompatible
	k8s.io/client-go v0.17.0
	sigs.k8s.io/kustomize/cmd/config v0.0.9
	sigs.k8s.io/kustomize/cmd/kubectl v0.0.3
	sigs.k8s.io/kustomize/cmd/resource v0.0.2
	sigs.k8s.io/kustomize/kyaml v0.0.8
)

exclude sigs.k8s.io/kustomize/pseudo/k8s v0.1.0
