// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build !tamago

package snvs

import (
	"crypto/aes"
	"errors"
)

func Encrypt(input []byte, key []byte, iv []byte) (output []byte, err error) {
	if len(input) < aes.BlockSize {
		return nil, errors.New("invalid length")
	}

	return encryptCTR(key, iv, input)
}

func Decrypt(input []byte, diversifier []byte) (output []byte, err error) {
	return nil, errors.New("not implemented")
}
