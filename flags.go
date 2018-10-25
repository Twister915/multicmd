package main

import "flag"

var (
	ThreadsFlag = flag.Int("threads", -1, "The number of threads to use (<1 means same as # of inputs)")
	AsyncFlag   = flag.Bool("async", true, "Whether or not to run the commands in parallel")
	QuietFlag   = flag.Bool("quiet", true, "Only print output from command")
	KillAllFlag = flag.Bool("kill-all", true, "If any command fails, kill all commands")
	TimeoutFlag = flag.Duration("timeout", 0, "timeout for ecah command")
)
