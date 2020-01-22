package complete

import (
	"flag"
)

// Complete default command line flag set defined by the standard library.
func CommandLine() {
	Complete(flag.CommandLine.Name(), FlagSet(flag.CommandLine))
}

// FlagSet returns a completer for a given standard library `flag.FlagSet`. It completes flag names,
// and additionally completes value if the `flag.Value` implements the `Predicate` interface.
func FlagSet(flags *flag.FlagSet) Completer {
	return (*flagSet)(flags)
}

type flagSet flag.FlagSet

func (fs *flagSet) SubCmdList() []string { return nil }

func (fs *flagSet) SubCmdGet(cmd string) Completer { return nil }

func (fs *flagSet) FlagList() []string {
	var flags []string
	(*flag.FlagSet)(fs).VisitAll(func(f *flag.Flag) {
		flags = append(flags, f.Name)
	})
	return flags
}

func (fs *flagSet) FlagGet(name string) Predictor {
	f := (*flag.FlagSet)(fs).Lookup(name)
	if f == nil {
		return nil
	}
	p, ok := f.Value.(Predictor)
	if !ok {
		return PredictFunc(func(string) []string { return []string{""} })
	}
	return p
}

func (fs *flagSet) ArgsGet() Predictor { return nil }
