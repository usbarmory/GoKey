// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package u2f

import (
	"encoding/binary"

	"github.com/f-secure-foundry/armoryctl/atecc608a"
)

const (
	counterCmd = 0x24
	read       = 0
	increment  = 1
	// Counter KeyID, #1 is used as it is never attached to any key.
	keyID = 0x01
)

// ATECC608A monotonic counter
type Counter struct {}

func (counter *Counter) Init() (err error) {
	_, err = atecc608a.SelfTest()
	return
}

func (counter *Counter) cmd(mode byte) (cnt uint32, err error) {
	res, err := atecc608a.ExecuteCmd(counterCmd, [1]byte{mode}, [2]byte{keyID, 0x00}, nil)

	if err != nil {
		return
	}

	return binary.LittleEndian.Uint32(res), nil
}

// Increment increases the ATECC608A monotonic counter in slot <1> (not attached to any key).
func (counter *Counter) Increment(_ []byte, _ []byte, _ []byte) (cnt uint32, err error) {
	return counter.cmd(increment)
}

// Counter reads the ATECC608A monotonic counter in slot <1> (not attached to any key).
func (counter *Counter) Read() (cnt uint32, err error) {
	return counter.cmd(read)
}
