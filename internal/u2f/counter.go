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
	"log"
	"time"

	"github.com/f-secure-foundry/armoryctl/atecc608a"
)

const (
	counterCmd = 0x24
	read       = 0
	increment  = 1
	// Counter KeyID, #1 is used as it is never attached to any key.
	keyID = 0x01
	// user presence timeout in seconds
	timeout = 10
)

// ATECC608A monotonic counter
type Counter struct {
	presence chan bool
}

// Init initializes the U2F counter function. A channel can be passed to
// receive user presence notifications, if nil user presence is automatically
// assumed.
func (c *Counter) Init(presence chan bool) (err error) {
	c.presence = presence
	_, err = atecc608a.SelfTest()
	return
}

func (c *Counter) cmd(mode byte) (cnt uint32, err error) {
	res, err := atecc608a.ExecuteCmd(counterCmd, [1]byte{mode}, [2]byte{keyID, 0x00}, nil)

	if err != nil {
		return
	}

	return binary.LittleEndian.Uint32(res), nil
}

// Increment increases the ATECC608A monotonic counter in slot <1> (not attached to any key).
func (c *Counter) Increment(appID []byte, challenge []byte, keyHandle []byte) (cnt uint32, err error) {
	log.Printf("U2F increment appId:%x challenge:%x keyHandle:%x", appID, challenge, keyHandle)

	return c.cmd(increment)
}

// Read reads the ATECC608A monotonic counter in slot <1> (not attached to any key).
func (c *Counter) Read() (cnt uint32, err error) {
	return c.cmd(read)
}

// UserPresence verifies the user presence.
func (c *Counter) UserPresence() bool {
	if c.presence == nil {
		return true
	}

	log.Printf("U2F request for user presence, issue `p` command to confirm")

	select {
	case <-c.presence:
		return true
	case <-time.After(timeout * time.Second):
		return false
	}
}
