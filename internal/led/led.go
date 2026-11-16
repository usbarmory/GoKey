// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build tamago && arm

package led

import (
	"runtime"
	"time"

	"github.com/usbarmory/armoryctl/led"
)

const interval = 200 * time.Millisecond

// Blink blinks an LED by name.
func Blink(name string, done chan bool) {
	var on bool

	for {
		select {
		case <-done:
			led.Set(name, false)
			return
		default:
		}

		on = !on
		led.Set(name, on)

		runtime.Gosched()
		time.Sleep(interval)
	}
}

// Set turns on/off an LED by name.
func Set(name string, on bool) (err error) {
	return led.Set(name, on)
}
