package arg

import (
	"strings"

	"github.com/posener/complete/v2/internal/tokener"
)

// Arg is typed a command line argument.
type Arg struct {
	Text      string
	Completed bool
	Parsed
}

// Parsed contains information about the argument.
type Parsed struct {
	Flag     string
	HasFlag  bool
	Value    string
	Dashes   string
	HasValue bool
}

// Parse parses a typed command line argument list, and returns a list of arguments.
func Parse(line string) []Arg {
	var args []Arg
	for {
		arg, after := next(line)
		if arg.Text != "" {
			args = append(args, arg)
		}
		line = after
		if line == "" {
			break
		}
	}
	return args
}

// next returns the first argument in the line and the rest of the line.
func next(line string) (arg Arg, after string) {
	defer arg.parse()
	// Start and end of the argument term.
	var start, end int
	// Stack of quote marks met during the paring of the argument.
	var token tokener.Tokener
	// Skip prefix spaces.
	for start = 0; start < len(line); start++ {
		token.Visit(line[start])
		if !token.LastSpace() {
			break
		}
	}
	// If line is only spaces, return empty argument and empty leftovers.
	if start == len(line) {
		return
	}

	for end = start + 1; end < len(line); end++ {
		token.Visit(line[end])
		if token.LastSpace() {
			arg.Completed = true
			break
		}
	}
	arg.Text = line[start:end]
	if !arg.Completed {
		return
	}
	start2 := end

	// Skip space after word.
	for start2 < len(line) {
		token.Visit(line[start2])
		if !token.LastSpace() {
			break
		}
		start2++
	}
	after = line[start2:]
	return
}

// parse a flag from an argument. The flag can have value attached when it is given in the
// `-key=value` format.
func (a *Arg) parse() {
	if len(a.Text) == 0 {
		return
	}

	// A pure value, no flag.
	if a.Text[0] != '-' {
		a.Value = a.Text
		a.HasValue = true
		return
	}

	// Seprate the dashes from the flag name.
	dahsI := 1
	if len(a.Text) > 1 && a.Text[1] == '-' {
		dahsI = 2
	}
	a.Dashes = a.Text[:dahsI]
	a.HasFlag = true
	a.Flag = a.Text[dahsI:]

	// Empty flag
	if a.Flag == "" {
		return
	}
	// Third dash or empty flag with equal is forbidden.
	if a.Flag[0] == '-' || a.Flag[0] == '=' {
		a.Parsed = Parsed{}
		return
	}
	// The flag is valid.

	// Check if flag has a value.
	if equal := strings.IndexRune(a.Flag, '='); equal != -1 {
		a.Flag, a.Value = a.Flag[:equal], a.Flag[equal+1:]
		a.HasValue = true
		return
	}

}
