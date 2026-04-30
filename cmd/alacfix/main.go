package main

import (
	"flag"
	"fmt"
	"os"

	"main/utils/alacfix"
)

func main() {
	var inPlace bool
	flag.BoolVar(&inPlace, "i", false, "modify file in place")
	flag.BoolVar(&inPlace, "in-place", false, "modify file in place")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  alacfix <input.m4a> <output.m4a>")
		fmt.Fprintln(os.Stderr, "  alacfix -i <input.m4a>")
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

	if err := alacfix.Run(input, output); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
