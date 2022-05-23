// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package u2f

import (
	"encoding/binary"
	"errors"
	"log"
	"runtime"
	"time"

	usbarmory "github.com/usbarmory/tamago/board/usbarmory/mk2"

	"github.com/usbarmory/armoryctl/atecc608"
	"github.com/usbarmory/armoryctl/led"
)

const (
	cntATECC = iota
	cntEMMC
)

const (
	read      = 0
	increment = 1

	// Counter KeyID, #1 is used as it is never attached to any key.
	keyID      = 0x01
	counterCmd = 0x24

	// When no HSM is present/supported (rev. γ) the counter value is saved
	// on the internal eMMC.
	//
	// The value is placed right before the Program Image offset (0x400) in
	// an area reserved for NXP optional Secondary Image Table
	// (0x200-0x400) but not used by the table itself.
	counterLBA    = 1
	counterOffset = 512 - 4

	// user presence timeout in seconds
	timeout = 10
)

// Counter represents an ATECC608A based monotonic counter instance.
type Counter struct {
	kind     int
	uid      []byte
	presence chan bool
}

// Init initializes an ATECC608A backed U2F counter. A channel can be passed to
// receive user presence notifications, if nil user presence is automatically
// assumed.
func (c *Counter) Init(presence chan bool) (err error) {
	switch usbarmory.Model() {
	case "UA-MKII-β": // ATECC608A
		var info string

		if info, err = atecc608.Info(); err != nil {
			return
		}

		c.uid = []byte(info)
		c.kind = cntATECC
	case "UA-MKII-γ": // NXP SE050 present but not supported
		if err = usbarmory.MMC.Detect(); err != nil {
			return
		}

		cid := usbarmory.MMC.Info().CID
		c.uid = cid[:]
		c.kind = cntEMMC
	default:
		errors.New("could not detect hardware model")
	}

	c.presence = presence

	return
}

// Serial returns the unique identifier for the hardware implementing the
// counter.
func (c *Counter) Serial() []byte {
	return c.uid
}

func (c *Counter) counterCmd(mode byte) (cnt uint32, err error) {
	switch c.kind {
	case cntATECC:
		res, err := atecc608.ExecuteCmd(counterCmd, [1]byte{mode}, [2]byte{keyID, 0x00}, nil, true)

		if err != nil {
			return 0, err
		}

		return binary.LittleEndian.Uint32(res), nil
	case cntEMMC:
		card := usbarmory.MMC
		buf := make([]byte, card.Info().BlockSize)

		if err = card.ReadBlocks(counterLBA, buf); err != nil {
			return
		}

		cnt = binary.LittleEndian.Uint32(buf[counterOffset:])

		if mode == read {
			return
		}

		cnt += 1
		binary.LittleEndian.PutUint32(buf[counterOffset:], cnt)

		if err = card.WriteBlocks(counterLBA, buf); err != nil {
			return
		}
	}

	return
}

// Increment increases the ATECC608A monotonic counter in slot <1> (not attached to any key).
func (c *Counter) Increment(_ []byte, _ []byte, _ []byte) (cnt uint32, err error) {
	cnt, err = c.counterCmd(increment)

	if err != nil {
		log.Printf("U2F increment failed, %v", err)
		return
	}

	log.Printf("U2F increment, counter:%d", cnt)

	return
}

// Read reads the ATECC608A monotonic counter in slot <1> (not attached to any key).
func (c *Counter) Read() (cnt uint32, err error) {
	return c.counterCmd(read)
}

// UserPresence verifies the user presence.
func (c *Counter) UserPresence() (present bool) {
	if c.presence == nil {
		return true
	}

	var done = make(chan bool)
	go blink(done)

	log.Printf("U2F user presence request, type `p` within %ds to confirm", timeout)

	select {
	case <-c.presence:
		present = true
	case <-time.After(timeout * time.Second):
		log.Printf("U2F user presence request timed out")
	}

	done <- true

	if present {
		log.Printf("U2F user presence confirmed")
	}

	return
}

func blink(done chan bool) {
	var on bool

	for {
		select {
		case <-done:
			led.Set("white", false)
			return
		default:
		}

		on = !on
		led.Set("white", on)

		runtime.Gosched()
		time.Sleep(200 * time.Millisecond)
	}
}
