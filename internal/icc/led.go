// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build tamago && arm

package icc

import (
	"github.com/usbarmory/armoryctl/led"
)

// LED turns on/off an LED by name.
func LED(name string, on bool) (err error) {
	return led.Set(name, on)
}
