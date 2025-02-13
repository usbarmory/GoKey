// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build !tamago

package icc

import (
	"log"
)

func LED(name string, on bool) (err error) {
	log.Printf("LED: %s, %v", name, on)
	return
}
