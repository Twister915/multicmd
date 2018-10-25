package main

import (
	"context"
	"os"
	"os/signal"
)

func SignalContext(parent context.Context, signals ...os.Signal) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	cancel = once(cancel)

	c := make(chan os.Signal)
	closeC := once(func() { close(c) })

	go func() {
		defer closeC()
		defer signal.Stop(c)

		signal.Notify(c, signals...)

		<-c
		cancel()
	}()

	return ctx, func() {
		closeC()
		<-ctx.Done()
	}
}
