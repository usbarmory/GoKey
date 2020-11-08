// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package icc

import (
	"github.com/f-secure-foundry/armoryctl/led"
)

// LED turns on/off an LED by name.
func LED(name string, on bool) (err error) {
	return led.Set(name, on)
}
