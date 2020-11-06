// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package snvs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"errors"

	"github.com/f-secure-foundry/tamago/soc/imx6"
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

func decryptOFB(key []byte, iv []byte, input []byte) (output []byte, err error) {
	block, err := aes.NewCipher(key)

	if err != nil {
		return
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(iv)

	if len(input) < mac.Size() {
		return nil, errors.New("invalid length for decrypt")
	}

	inputMac := input[len(input)-mac.Size():]
	mac.Write(input[0 : len(input)-mac.Size()])

	if !hmac.Equal(inputMac, mac.Sum(nil)) {
		return nil, errors.New("invalid HMAC")
	}

	stream := cipher.NewOFB(block, iv)
	output = make([]byte, len(input)-mac.Size())

	stream.XORKeyStream(output, input[0:len(input)-mac.Size()])

	return
}

// DecryptOFB performs symmetric AES decryption using AES-256-OFB. The
// ciphertext format is expected to have the initialization vector prepended
// and an HMAC for authentication appended: `iv (16 bytes) || ciphertext ||
// hmac (32 bytes)`.
//
// The key is derived, with a diversifier, from the SNVS device-specific OTPMK
// secret.
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
	output, err = decryptOFB(key, iv, input[aes.BlockSize:])

	return
}
