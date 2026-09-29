package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sijiling/logpilot/internal/eval"
)

func main() {
	casesPath := flag.String("cases", "testdata/eval/cases.jsonl", "path to JSONL evaluation cases")
	resultsPath := flag.String("results", "", "path to JSONL observed results")
	flag.Parse()
	if *resultsPath == "" {
		fmt.Fprintln(os.Stderr, "-results is required")
		os.Exit(2)
	}
	casesFile, err := os.Open(*casesPath)
	if err != nil {
		fail(err)
	}
	defer casesFile.Close()
	resultsFile, err := os.Open(*resultsPath)
	if err != nil {
		fail(err)
	}
	defer resultsFile.Close()
	cases, err := eval.LoadCases(casesFile)
	if err != nil {
		fail(err)
	}
	results, err := eval.LoadResults(resultsFile)
	if err != nil {
		fail(err)
	}
	metrics, err := eval.Calculate(cases, results)
	if err != nil {
		fail(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(metrics); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
