package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Runner struct {
	c chan Task
	wg *sync.WaitGroup

	cancel func()
	cancelAll bool

	taskTimeout time.Duration

	quiet bool

	outLock sync.Mutex
}

func NewRunner(quiet, cancelAll bool, taskTimeout time.Duration) *Runner {
	return &Runner{c: make(chan Task), wg: new(sync.WaitGroup), quiet: quiet, cancelAll: cancelAll, taskTimeout: taskTimeout}
}

func (r *Runner) Run(ctx context.Context, workers int) (allOK bool) {
	var err error
	defer func() {
		if r.cancel != nil {
			r.cancel()
		}
	}()

	defer func() {
		if err == nil {
			err = ctx.Err()
		}

		if err == nil {
			allOK = true
		}
	}()
	defer r.wg.Wait()
	defer close(r.c)

	handleError := func(e error, msg string) {
		err = e
		fmt.Fprintf(os.Stderr, "ERROR %s: %s\n", msg, e)
		r.cancel()
	}

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = once(func() {
		cancel()
	})
	r.startWorkers(ctx, workers)

	commandFormat, hasTarget := getCommandFormat()
	if !hasTarget {
		handleError(errors.New("no target specified in command (use {})"), "setting up command")
		return
	}

	inputs := make(chan input)
	go readInputs(ctx, inputs)

	i := 1
	for {
		select {
		case input, ok := <-inputs:
			if !ok {
				return
			}

			if input.err != nil {
				handleError(input.err, "reading input from stdin")
				return
			}

			select {
			case r.c <- Task{Target: input.v, CommandFormat: commandFormat, I: i}:
				i++

			case <-ctx.Done():
				return
			}
		}
	}
}

func (r *Runner) startWorkers(ctx context.Context, workers int) {
	r.wg.Add(workers)

	for i := 0; i < workers; i++ {
		go r.worker(ctx)
	}
}

func (r *Runner) worker(ctx context.Context) {
	defer r.wg.Done()

	for t := range r.c {
		func() {
			c := ctx
			cancel := func() {}
			if r.taskTimeout > 0 {
				c, cancel = context.WithTimeout(c, r.taskTimeout)
			}

			defer cancel()
			r.activate(c, t)
		}()
	}
}

type input struct {
	err error
	v string
}

func readInputs(ctx context.Context, c chan input) {
	defer close(c)
	stat, err := os.Stdin.Stat()
	if err != nil {
		c <- input{err: err}
		return
	}

	if (stat.Mode() & os.ModeCharDevice) != 0 {
		c <- input{err: fmt.Errorf("stdin has no data provided to it")}
		return
	}

	reader := bufio.NewReader(os.Stdin)

	var builder bytes.Buffer
	for err == nil {
		var ch rune
		ch, _, err = reader.ReadRune()

		if err == nil {
			switch ch {
			case '\n', ' ':
				if builder.Len() > 0 {
					select {
					case c <- input{v: builder.String()}:
						builder.Reset()

					case <-ctx.Done():
						err = ctx.Err()
					}
				}
			default:
				builder.WriteRune(ch)
			}
		}
	}

	if err == io.EOF {
		err = nil
	}

	if err != nil {
		c <- input{err: err}
	}
}

func getCommandFormat() (func(string) []string, bool) {
	args := flag.Args()
	var toReplace []int
	for i, a := range args {
		if a == "{}" {
			toReplace = append(toReplace, i)
		}
	}

	return func(target string) (out []string) {
		k := 0
		for i, a := range args {
			if len(toReplace) > k && toReplace[k] == i {
				out = append(out, target)
				k++
			} else {
				out = append(out, a)
			}
		}

		return
	}, len(toReplace) > 0
}
