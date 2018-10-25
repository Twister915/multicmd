package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"syscall"
)

// Takes a list of targets through standard input
// Takes a command format in args
// Takes flags for options
func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	runner := NewRunner(*QuietFlag, *KillAllFlag, *TimeoutFlag)
	ctx := context.Background()
	ctx, cancel := SignalContext(ctx, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	workers := *ThreadsFlag
	if workers < 1 {
		workers = runtime.NumCPU()
	}

	if !*AsyncFlag {
		workers = 1
	}

	if runner.Run(ctx, workers) {
		fmt.Fprintf(os.Stderr, "SUCCESS [ALL]\n")
	}
}