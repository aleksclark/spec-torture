// Command arp-conformance runs the ARP conformance suite against the reference
// server (in-process, mock backend) and writes a markdown report. It exits
// non-zero if any required-severity check fails.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/aleksclark/spec-torture/arp/conformance"
)

func main() {
	out := flag.String("out", "", "write the markdown report to this file (default: stdout only)")
	title := flag.String("title", "ARP Reference Implementation — gRPC Conformance", "report title")
	flag.Parse()

	env, err := conformance.Start(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "start: %v\n", err)
		os.Exit(2)
	}
	defer env.Stop()

	results, err := conformance.Run(env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		os.Exit(2)
	}

	md := conformance.Markdown(*title, results)
	fmt.Print(md)
	if *out != "" {
		if err := os.WriteFile(*out, []byte(md), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write report: %v\n", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
	}

	if fails := conformance.RequiredFailures(results); len(fails) > 0 {
		fmt.Fprintf(os.Stderr, "FAILED required checks: %v\n", fails)
		os.Exit(1)
	}
}
