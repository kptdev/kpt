module kpt.dev

go 1.12

require (
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/olekukonko/tablewriter v0.0.1
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.4.0
	gotest.tools v2.2.0+incompatible
	k8s.io/apimachinery v0.0.0-20190830154629-f1cbc0c8ff07
	lib.kpt.dev v0.0.0
	sigs.k8s.io/kustomize/v3 v3.1.1-0.20190830180857-4ebad27d7a6d
)

replace lib.kpt.dev => ../lib/
