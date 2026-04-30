package main

import (
	"flag"
	"fmt"
	"os"

	"main/utils/alacfix"
)

func main() {
	var inPlace, force bool
	flag.BoolVar(&inPlace, "i", false, "modify file in place")
	flag.BoolVar(&inPlace, "in-place", false, "modify file in place")
	flag.BoolVar(&force, "f", false, "always write output even if no patches applied")
	flag.BoolVar(&force, "force", false, "always write output even if no patches applied")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  alacfix [-f] <input.m4a> <output.m4a>")
		fmt.Fprintln(os.Stderr, "  alacfix [-f] -i <input.m4a>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  -i, --in-place   modify file in place")
		fmt.Fprintln(os.Stderr, "  -f, --force      always write output even if no patches applied")
	}
	flag.Parse()

	args := flag.Args()

	var input, output string
	switch {
	case inPlace && len(args) == 1:
		input = args[0]
	case !inPlace && len(args) == 2:
		input, output = args[0], args[1]
	default:
		flag.Usage()
		os.Exit(1)
	}

	if err := alacfix.Run(input, force, output); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
