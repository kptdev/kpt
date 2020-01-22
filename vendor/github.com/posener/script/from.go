package script

import (
	"os"
	"strings"
)

// Stdin starts a stream from stdin.
func Stdin() Stream {
	return From("stdin", os.Stdin)
}

// Echo writes to stdout.
//
// Shell command: `echo <s>`
func Echo(s string) Stream {
	return From("echo", strings.NewReader(s+"\n"))
}
