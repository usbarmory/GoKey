// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build !tamago

package led

import (
	"log"
	"runtime"
	"time"
)

const interval = 1 * time.Second

// Blink blinks an LED by name.
func Blink(name string, done chan bool) {
	var on bool

	for {
		select {
		case <-done:
			Set(name, false)
			return
		default:
		}

		on = !on
		Set(name, on)

		runtime.Gosched()
		time.Sleep(interval)
	}
}

// Set turns on/off an LED by name.
func Set(name string, on bool) (err error) {
	log.Printf("LED: %s, %v", name, on)
	return
}
