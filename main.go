package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"
)

func main() {
	input := flag.String("input", "", "path to input CSV file")
	output := flag.String("output", "", "output directory for report files")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: aggregator --input <file> --output <dir>")
		os.Exit(1)
	}

	start := time.Now()

	if err := run(*input, *output); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(start)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(os.Stderr, "completed in %s, heap alloc: %.1f MB, total alloc: %.1f MB\n",
		elapsed, float64(m.HeapAlloc)/1e6, float64(m.TotalAlloc)/1e6)
}
