package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-semantic-regression-bisector/internal/bisector"
)

func main() {
	manifestPath := flag.String("manifest", "", "path to the pinned bisect manifest JSON")
	observationsPath := flag.String("observations", "", "path to supplied observation receipts in NDJSON")
	outputDir := flag.String("out", "output", "caller-owned output directory")
	flag.Parse()
	if *manifestPath == "" || *observationsPath == "" {
		fmt.Fprintln(os.Stderr, "--manifest and --observations are required")
		os.Exit(2)
	}
	result, err := bisector.RunFiles(*manifestPath, *observationsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := bisector.WriteOutputs(*outputDir, result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("decision=%s class=%s outputs=6\n", result.Boundary.Decision, result.Boundary.Class)
}
