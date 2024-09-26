// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build tamago && arm

package icc

import (
	"time"

	"github.com/usbarmory/tamago/soc/nxp/imx6ul"
)

// SleepTimeout is the time duration to fall back to low power mode after the
// token is awakened by a command.
const SleepTimeout = 60 * time.Second

// Wake sets the CPU speed to its highest frequency. After the duration defined
// in the `SleepTimeout` constant the CPU speed is restored to its lowest
// frequency.
func (card *Interface) Wake() {
	card.Lock()
	defer card.Unlock()

	if card.awake {
		return
	}

	_ = imx6ul.SetARMFreq(imx6ul.FreqMax)
	card.awake = true

	time.AfterFunc(SleepTimeout, func() {
		card.Lock()
		defer card.Unlock()

		_ = imx6ul.SetARMFreq(imx6ul.FreqLow)
		card.awake = false
	})
}
