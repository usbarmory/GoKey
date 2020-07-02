// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package icc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"github.com/f-secure-foundry/tamago/imx6"

	"github.com/hsanjuan/go-nfctype4/apdu"
)

func init() {
	// When running natively on i.MX6, and under secure boot, the built-in
	// Data Co-Processor (DCP) is used for AES key derivation.
	//
	// A device specific random 256-bit OTPMK key is fused in each SoC at
	// manufacturing time, this key is unreadable and can only be used by
	// the DCP for AES encryption/decryption of user data, through the
	// Secure Non-Volatile Storage (SNVS) companion block.
	//
	// This is leveraged to create the AES256 DO used by PSO:DEC and
	// PSO:ENC and to allow encrypted bundling of OpenPGP secret keys.
	if imx6.Native && imx6.DCP.SNVS() {
		imx6.DCP.Init()
	}
}

func encipher(data []byte) (rapdu *apdu.RAPDU, err error) {
	return aesCBC(data, false)
}

func decipher(data []byte) (rapdu *apdu.RAPDU, err error) {
	return aesCBC(data, true)
}

func aesCBC(data []byte, decrypt bool) (rapdu *apdu.RAPDU, err error) {
	if !imx6.Native || !imx6.DCP.SNVS() {
		return CommandNotAllowed(), nil
	}

	iv := make([]byte, aes.BlockSize)
	key, err := imx6.DCP.DeriveKey(RID, iv, -1)

	if err != nil {
		return
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

func encrypt(input []byte, diversifier []byte) (output []byte, err error) {
	// It is advised to use only deterministic input data for key
	// derivation, therefore we use the empty allocated IV before it being
	// filled.
	iv := make([]byte, aes.BlockSize)
	key, err := imx6.DCP.DeriveKey(diversifier, iv, -1)

	if err != nil {
		return
	}
	_, err = io.ReadFull(rand.Reader, iv)

	if err != nil {
		return
	}

	output, err = EncryptOFB(key, iv, input)

	return
}

func Decrypt(input []byte, diversifier []byte) (output []byte, err error) {
	// It is advised to use only deterministic input data for key
	// derivation, therefore we use the empty allocated IV before it being
	// filled.
	iv := make([]byte, aes.BlockSize)
	key, err := imx6.DCP.DeriveKey(diversifier, iv, -1)

	if err != nil {
		return
	}

	if len(input) < aes.BlockSize {
		return nil, errors.New("invalid length for decrypt")
	}

	iv = input[0:aes.BlockSize]
	output, err = DecryptOFB(key, iv, input[aes.BlockSize:])

	return
}
