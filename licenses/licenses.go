package licenses

import _ "embed"

var (
	//go:embed kpt.txt
	AllOSSLicense string
)
