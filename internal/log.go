// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package gokey

import (
	"log"
	"os"
)

// initialized at compile time (see Makefile)
var Build string
var Revision string

var Banner string

func init() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)
}
