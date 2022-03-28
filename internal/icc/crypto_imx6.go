// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package icc

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/usbarmory/tamago/soc/imx6/dcp"

	"github.com/hsanjuan/go-nfctype4/apdu"
)

func encipher(data []byte) (rapdu *apdu.RAPDU, err error) {
	return aesCBC(data, false)
}

func decipher(data []byte) (rapdu *apdu.RAPDU, err error) {
	return aesCBC(data, true)
}

func aesCBC(data []byte, decrypt bool) (rapdu *apdu.RAPDU, err error) {
	iv := make([]byte, aes.BlockSize)
	key, err := dcp.DeriveKey(RID, iv, -1)

	if err != nil {
		return CommandNotAllowed(), nil
	}

	if len(data)%aes.BlockSize != 0 {
		return WrongData(), nil
	}

	block, err := aes.NewCipher(key)

	if err != nil {
		return
	}

	var mode cipher.BlockMode

	if decrypt {
		mode = cipher.NewCBCDecrypter(block, iv)
	} else {
		mode = cipher.NewCBCEncrypter(block, iv)
	}

	mode.CryptBlocks(data, data)

	return CommandCompleted(data), nil
}
