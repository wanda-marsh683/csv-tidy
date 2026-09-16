package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	lenient := flag.Bool("lenient", false, "repair messy input instead of rejecting it (pads/truncates ragged rows, trims whitespace, fixes invalid UTF-8)")
	delimiter := flag.String("delimiter", ",", "field delimiter (single character)")
	flag.Parse()

	if len(*delimiter) != 1 {
		fmt.Fprintln(os.Stderr, "csv-tidy: --delimiter must be exactly one character")
		os.Exit(2)
	}

	input := os.Stdin
	switch args := flag.Args(); len(args) {
	case 0:
		// read from stdin
	case 1:
		f, err := os.Open(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "csv-tidy: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		input = f
	default:
		fmt.Fprintln(os.Stderr, "usage: csv-tidy [--lenient] [--delimiter ,] [file]")
		os.Exit(2)
	}

	n := &Normalizer{Delimiter: rune((*delimiter)[0]), Lenient: *lenient}
	if err := n.Process(input, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "csv-tidy: %v\n", err)
		os.Exit(1)
	}
}
