// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build !tamago

package icc

import (
	"log"
)

func LED(name string, on bool) (err error) {
	log.Printf("LED: %s, %v", name, on)
	return
}
