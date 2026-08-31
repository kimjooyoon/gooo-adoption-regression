package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-adoption-regression/internal/regression"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintln(os.Stderr, "usage: gooo-adoption-regression run --source PATH --contract PATH --corpus PATH --ci-runtime PATH --output-dir ABSOLUTE_EMPTY_DIR")
		os.Exit(2)
	}
	flags := flag.NewFlagSet("run", flag.ExitOnError)
	source := flags.String("source", "", "canonical .gooo source")
	contract := flags.String("contract", "", "denominator contract")
	corpus := flags.String("corpus", "", "canonical case corpus")
	runtime := flags.String("ci-runtime", "", "CI runtime receipt")
	outputDir := flags.String("output-dir", "", "caller-owned empty absolute output directory")
	if err := flags.Parse(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for name, value := range map[string]string{
		"source": *source,
		"contract": *contract,
		"corpus": *corpus,
		"ci-runtime": *runtime,
		"output-dir": *outputDir,
	} {
		if value == "" {
			fmt.Fprintf(os.Stderr, "missing --%s\n", name)
			os.Exit(2)
		}
	}
	if err := regression.Run(*source, *contract, *corpus, *runtime, *outputDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
