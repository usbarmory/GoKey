// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build tamago && arm

package snvs

import (
	"crypto/aes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"errors"

	"filippo.io/keygen"
	"golang.org/x/crypto/hkdf"

	"github.com/usbarmory/tamago/soc/nxp/imx6ul"
	"github.com/usbarmory/tamago/soc/nxp/snvs"
)

const diversifierDev = "GoKeySNVSDeviceK"

func init() {
	if !imx6ul.SNVS.Available() {
		return
	}

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
	imx6ul.DCP.Init()

	// Disable ARM debug operations
	imx6ul.Debug(false)

	imx6ul.SNVS.SetPolicy(
		snvs.SecurityPolicy{
			Clock:             true,
			Temperature:       true,
			Voltage:           true,
			SecurityViolation: true,
			HardFail:          true,
		},
	)
}

// DeviceKey derives a device key, uniquely and deterministically generated for
// this SoC for attestation purposes.
func DeviceKey() (deviceKey *ecdsa.PrivateKey, err error) {
	iv := make([]byte, aes.BlockSize)
	key, err := imx6ul.DCP.DeriveKey([]byte(diversifierDev), iv, -1)

	if err != nil {
		return
	}

	salt := imx6ul.UniqueID()
	r := hkdf.New(sha256.New, key, salt[:], nil)

	return keygen.ECDSALegacy(elliptic.P256(), r)
}

// Encrypt performs symmetric AES encryption using AES-256-CTR. The
// initialization vector is prepended to the encrypted file, the HMAC for
// authentication is appended: `iv (16 bytes) || ciphertext || hmac (32
// bytes)`.
//
// The key is derived, with a diversifier, from the SNVS device-specific OTPMK
// secret.
func Encrypt(input []byte, diversifier []byte, iv []byte) (output []byte, err error) {
	// It is advised to use only deterministic input data for key
	// derivation.
	kdfIV := make([]byte, aes.BlockSize)
	key, err := imx6ul.DCP.DeriveKey(diversifier, kdfIV, -1)

	if err != nil {
		return
	}

	if len(input) < aes.BlockSize {
		return nil, errors.New("invalid length")
	}

	return encryptCTR(key, iv, input)
}

// Decrypt performs symmetric AES decryption using AES-256-CTR. The
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
	key, err := imx6ul.DCP.DeriveKey(diversifier, iv, -1)

	if err != nil {
		return
	}

	if len(input) < aes.BlockSize {
		return nil, errors.New("invalid length")
	}

	iv = input[0:aes.BlockSize]

	return decryptCTR(key, iv, input[aes.BlockSize:])
}
