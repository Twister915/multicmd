package main

import (
	"io"
	"sync"
)

func slurp(wg *sync.WaitGroup, to *string, source io.Reader) error {
	defer wg.Done()
	var buf [1024]byte
	var n int
	var err error

	for err == nil {
		n, err = source.Read(buf[:])
		if n > 0 {
			*to = (*to) + string(buf[:n])
		}
	}

	if err == io.EOF {
		err = nil
	}

	return err
}
