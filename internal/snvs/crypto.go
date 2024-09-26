// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package snvs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
)

func encryptCTR(key []byte, iv []byte, input []byte) (output []byte, err error) {
	block, err := aes.NewCipher(key)

	if err != nil {
		return
	}

	output = iv

	mac := hmac.New(sha256.New, key)
	mac.Write(iv)

	stream := cipher.NewCTR(block, iv)
	output = append(output, make([]byte, len(input))...)

	stream.XORKeyStream(output[len(iv):], input)
	mac.Write(output[len(iv):])

	output = append(output, mac.Sum(nil)...)

	return
}

func decryptCTR(key []byte, iv []byte, input []byte) (output []byte, err error) {
	block, err := aes.NewCipher(key)

	if err != nil {
		return
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(iv)

	if len(input) < mac.Size() {
		return nil, errors.New("invalid length")
	}

	inputMac := input[len(input)-mac.Size():]
	mac.Write(input[0 : len(input)-mac.Size()])

	if !hmac.Equal(inputMac, mac.Sum(nil)) {
		return nil, errors.New("invalid HMAC")
	}

	stream := cipher.NewCTR(block, iv)
	output = make([]byte, len(input)-mac.Size())

	stream.XORKeyStream(output, input[0:len(input)-mac.Size()])

	return
}
