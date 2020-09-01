// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package gokey

import (
	"github.com/f-secure-foundry/tamago/soc/imx6"
)

func reboot() {
	imx6.Reboot()
}
