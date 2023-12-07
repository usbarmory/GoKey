// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build !tamago
// +build !tamago

package snvs

import (
	"errors"
)

func Decrypt(input []byte, diversifier []byte) (output []byte, err error) {
	return nil, errors.New("not implemented")
}
