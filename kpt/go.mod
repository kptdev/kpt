module kpt.dev/kpt

go 1.12

require (
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/go-errors/errors v1.0.1
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/olekukonko/tablewriter v0.0.1
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.4.0
	gotest.tools v2.2.0+incompatible
	lib.kpt.dev v0.0.0
	sigs.k8s.io/kustomize/kyaml v0.0.0-20191113193121-24f3d7556f20
	sigs.k8s.io/kustomize/v3 v3.2.0
)

replace lib.kpt.dev => ../lib/
