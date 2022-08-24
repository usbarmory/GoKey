// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package usb

import (
	"sync"

	"github.com/usbarmory/tamago/soc/nxp/imx6ul"
)

var (
	mux sync.Mutex
	cnt int
)

func wake() {
	mux.Lock()
	defer mux.Unlock()

	if cnt == 0 {
		_ = imx6ul.SetARMFreq(imx6ul.FreqMax)
	}

	cnt += 1
}

func idle() {
	mux.Lock()
	defer mux.Unlock()

	cnt -= 1

	if cnt == 0 {
		_ = imx6ul.SetARMFreq(imx6ul.FreqLow)
	}
}
