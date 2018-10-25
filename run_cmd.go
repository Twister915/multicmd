package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

func (r *Runner) activate(ctx context.Context, task Task) {
	var err error
	var stderr, stdout string

	commandParts := task.CommandFormat(task.Target)
	defer func() {
		if err != nil && err != context.Canceled {
			r.handleErr(err, task.I, fmt.Sprintf("command failed due to error"), stdout, stderr, commandParts)
		}
	}()

	exe, args := commandParts[0], commandParts[1:]

	cmd := exec.CommandContext(ctx, exe, args...)
	var outPipe, errPipe io.ReadCloser

	if outPipe, err = cmd.StdoutPipe(); err != nil {
		return
	}

	if errPipe, err = cmd.StderrPipe(); err != nil {
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go slurp(&wg, &stderr, errPipe)
	go slurp(&wg, &stdout, outPipe)

	r.writeCommandMsg("BEGIN", task.I, commandParts, "", "")
	if err = cmd.Start(); err != nil {
		return
	}

	var exitCode int

	if err = cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if sysStatus, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = sysStatus.ExitStatus()
			} else {
				return
			}
		} else {
			return
		}
	}

	wg.Wait()

	if exitCode != 0 {
		r.handleErr(err, task.I, fmt.Sprintf("exit code = %d", exitCode), stdout, stderr, commandParts)
		err = nil
		return
	}

	r.handleSuccess(task.I, stdout, stderr, commandParts)
	return
}

func (r *Runner) handleErr(err error, i int, msg, stdout, stderr string, command []string) {
	if r.cancelAll {
		r.cancel()
	}

	r.writeCommandMsg(fmt.Sprintf("FAILURE (%s)", err.Error()), i, command, stdout, stderr)
}

func (r *Runner) handleSuccess(i int, stdout, stderr string, command []string) {
	r.writeCommandMsg("SUCCESS", i, command, stdout, stderr)
}

func (r *Runner) writeCommandMsg(msg string, i int, command []string, stdout, stderr string) {
	var buf bytes.Buffer
	defer func() {
		if buf.Len() != 0 {
			r.write(buf.String())
		}
	}()
	fmt.Fprintf(&buf, "%s [%d]: %s\n", msg, i, strings.Join(command, " "))

	stdout = strings.TrimSpace(stdout)
	stderr = strings.TrimSpace(stderr)

	if !r.quiet {
		//fmt.Fprintf(os.Stderr, "\n   output (stderr):\n%s\n\n   output (stdout):\n%s\n", indent(stderr, "   "), indent(stdout, "   "))
		if stderr != "" {
			fmt.Fprintf(&buf, "output (stderr):\n%s\n", indent(stderr, "> "))
		}

		if stdout != "" {
			fmt.Fprintf(&buf, "output (stdout):\n%s\n", indent(stdout, "> "))
		}

		if stdout != "" || stderr != "" {
			fmt.Fprintf(&buf, "\n")
		}
	}
}

func (r *Runner) write(v string) {
	r.outLock.Lock()
	defer r.outLock.Unlock()

	fmt.Fprint(os.Stderr, v)
}