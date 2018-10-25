package main

import "sync/atomic"

func once(f func()) func() {
	k := new(int32)
	return func() {
		if atomic.CompareAndSwapInt32(k, 0, 1) {
			f()
		}
	}
}
