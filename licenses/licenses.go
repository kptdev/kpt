package licenses

// blank import is required as per
// https://pkg.go.dev/embed#hdr-Strings_and_Bytes
import _ "embed"

var (
	//go:embed kpt.txt
	AllOSSLicense string
)
