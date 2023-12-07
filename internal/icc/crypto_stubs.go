// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build !tamago
// +build !tamago

package icc

import (
	"errors"

	"github.com/hsanjuan/go-nfctype4/apdu"
)

func encipher(data []byte) (rapdu *apdu.RAPDU, err error) {
	return nil, errors.New("not implemented")
}

func decipher(data []byte) (rapdu *apdu.RAPDU, err error) {
	return nil, errors.New("not implemented")
}

func Decrypt(input []byte, diversifier []byte) (output []byte, err error) {
	return nil, errors.New("not implemented")
}
