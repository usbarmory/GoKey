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
	"fmt"
	"runtime"

	"github.com/f-secure-foundry/tamago/soc/imx6"
)

func init() {
	Banner = fmt.Sprintf("GoKey • %s/%s (%s) • %s %s",
		runtime.GOOS, runtime.GOARCH, runtime.Version(),
		Revision, Build)

	Banner += fmt.Sprintf(" • %s", imx6.Model())
}
