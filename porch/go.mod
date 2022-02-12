// This is a generated file. Do not edit directly.

module github.com/GoogleContainerTools/kpt/porch

go 1.17

require (
	github.com/GoogleContainerTools/kpt/porch/apiserver v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.3.0
	k8s.io/apimachinery v0.23.3
	k8s.io/client-go v0.23.3
	k8s.io/klog/v2 v2.40.1
)

require (
	github.com/GoogleContainerTools/kpt/porch/api v0.0.0-00010101000000-000000000000 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kr/pretty v0.2.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/api v0.23.3 // indirect
	k8s.io/utils v0.0.0-20211208161948-7d6a63dca704 // indirect
	sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	github.com/GoogleContainerTools/kpt => ../.
	github.com/GoogleContainerTools/kpt/porch/api => ./api
	github.com/GoogleContainerTools/kpt/porch/apiserver => ./apiserver
	github.com/GoogleContainerTools/kpt/porch/controllers => ./controllers
	github.com/GoogleContainerTools/kpt/porch/engine => ./engine
	github.com/GoogleContainerTools/kpt/porch/func => ./func
	github.com/GoogleContainerTools/kpt/porch/repository => ./repository
)
