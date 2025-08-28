// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build tamago && arm

package icc

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/usbarmory/tamago/soc/nxp/imx6ul"

	"github.com/hsanjuan/go-nfctype4/apdu"
)

func encipher(data []byte) (rapdu *apdu.RAPDU, err error) {
	return aesCBC(data, false)
}

func decipher(data []byte) (rapdu *apdu.RAPDU, err error) {
	return aesCBC(data, true)
}

func aesCBC(data []byte, decrypt bool) (rapdu *apdu.RAPDU, err error) {
	if !imx6ul.SNVS.Available() {
		return CommandNotAllowed(), nil
	}

	iv := make([]byte, aes.BlockSize)
	key, err := imx6ul.DCP.DeriveKey(RID, iv, -1)

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
