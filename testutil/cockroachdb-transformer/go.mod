module kpt.dev/transformer

go 1.12

require (
	gopkg.in/yaml.v3 v3.0.0-20190924164351-c8b7dadae555
	kpt.dev v0.0.0
	lib.kpt.dev v0.0.0
)

replace (
	kpt.dev => ../../kpt/
	lib.kpt.dev => ../../lib/
)
