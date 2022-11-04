// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package main

import (
	_ "unsafe"

	"github.com/usbarmory/tamago/dma"
)

// Override usbarmory pkg ramSize and `mem` allocation, as having 3 USB
// endpoints (CDC, ICC, U2F) requires more than what the iRAM can handle.

//go:linkname ramSize runtime.ramSize
var ramSize uint = 0x10000000 - 0x100000 // 256MB - 1MB
var dmaStart uint = 0xa0000000 - 0x100000

// 1MB
var dmaSize = 0x100000

func init() {
	dma.Init(dmaStart, dmaSize)
}
