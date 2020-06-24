// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package icc

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"log"
	"math"
	"math/big"

	"github.com/hsanjuan/go-nfctype4/apdu"
	"github.com/keybase/go-crypto/openpgp/ecdh"
	"github.com/keybase/go-crypto/rsa"
)

// Diversifier for hardware key derivation (OpenPGP key wrapping).
const DiversifierPGP = "GoKeySNVSOpenPGP"

// Diversifier for hardware key derivation (SSH private key wrapping).
const DiversifierSSH = "GoKeySNVSOpenSSH"

const (
	// p65, 7.2.11 PSO: DECIPHER, OpenPGP application Version 3.4
	RSA_PADDING = 0x00
	AES_PADDING = 0x02
)

func padToKeySize(pub ecdsa.PublicKey, b []byte) []byte {
	k := (pub.Curve.Params().BitSize + 7) / 8
	if len(b) >= k {
		return b
	}
	bb := make([]byte, k)
	copy(bb[len(bb)-len(b):], b)
	return bb
}

// ComputeDigitalSignature implements
// p62, 7.2.10 PSO: COMPUTE DIGITAL SIGNATURE, OpenPGP application Version 3.4.
func (card *Interface) ComputeDigitalSignature(data []byte) (rapdu *apdu.RAPDU, err error) {
	var sig []byte

	if len(data) == 0 {
		return WrongData(), nil
	}

	subkey := card.Sig

	if subkey == nil || subkey.PrivateKey == nil {
		log.Printf("missing private key for PSO:COMPUTE DIGITAL SIGNATURE")
		return CardKeyNotSupported(), nil
	}

	if subkey.PrivateKey.Encrypted {
		return SecurityConditionNotSatisfied(), nil
	}

	if PW1_CDS_MULTI == 0 {
		defer card.Verify(PW_LOCK, PW1_CDS, nil)
	}

	switch privKey := subkey.PrivateKey.PrivateKey.(type) {
	case *rsa.PrivateKey:
		var hash crypto.Hash

		// p64, 7.2.10.2 DigestInfo for RSA, OpenPGP application Version 3.4

		// 19 bytes of DigestInfo + at least 32 bytes for smallest hash
		if len(data) < 19+32 {
			return WrongData(), nil
		}

		digest := data[19:]

		switch len(digest) {
		case 32:
			hash = crypto.SHA256
		case 48:
			hash = crypto.SHA384
		case 64:
			hash = crypto.SHA512
		default:
			return WrongData(), nil
		}

		sig, err = rsa.SignPKCS1v15(rand.Reader, privKey, hash, digest)
	case *ecdsa.PrivateKey:
		// OpenPGP uses ECDSA signatures in raw format
		r, s, e := ecdsa.Sign(rand.Reader, privKey, data)

		if e != nil {
			err = e
		} else {
			// https://tools.ietf.org/html/rfc7518#section-3.4
			//
			// "...adds zero-valued high-order padding bits when
			// needed to round the size up to a multiple of 8 bits;
			// thus, each 521-bit integer is represented using 528
			// bits in 66 octets."
			sig = append(sig, padToKeySize(privKey.PublicKey, r.Bytes())...)
			sig = append(sig, padToKeySize(privKey.PublicKey, s.Bytes())...)
		}
	default:
		log.Printf("invalid private key for PSO:COMPUTE DIGITAL SIGNATURE")
		return CardKeyNotSupported(), nil
	}

	if err != nil {
		log.Printf("PSO:COMPUTE DIGITAL SIGNATURE error, %v", err)
		return UnrecoverableError(), nil
	}

	log.Printf("PSO:CDS successful")
	card.digitalSignatureCounter += 1

	return CommandCompleted(sig), nil
}

// Decipher implements
// p65, 7.2.11 PSO: DECIPHER, OpenPGP application Version 3.4.
func (card *Interface) Decipher(data []byte) (rapdu *apdu.RAPDU, err error) {
	var plaintext []byte

	if len(data) < 1 {
		return WrongData(), nil
	}

	if data[0] == AES_PADDING {
		return decipher(data[1:])
	}

	subkey := card.Dec

	if subkey == nil || subkey.PrivateKey == nil {
		log.Printf("missing private key for PSO:DEC")
		return CardKeyNotSupported(), nil
	}

	if subkey.PrivateKey.Encrypted {
		return SecurityConditionNotSatisfied(), nil
	}

	switch privKey := subkey.PrivateKey.PrivateKey.(type) {
	case *rsa.PrivateKey:
		if data[0] != RSA_PADDING {
			log.Printf("invalid private key for PSO:DEC")
			return CardKeyNotSupported(), nil
		}

		plaintext, err = privKey.Decrypt(rand.Reader, data, nil)
	case *ecdh.PrivateKey:
		X := big.NewInt(0)
		Y := big.NewInt(0)

		if data[0] != DO_CIPHER {
			log.Printf("invalid private key for PSO:DEC")
			return CardKeyNotSupported(), nil
		}

		// p66, 7.2.11 PSO: DECIPHER, OpenPGP application Version 3.4
		pubKey := v(v(v(data, DO_CIPHER), DO_PUB_KEY), DO_EXT_PUB_KEY)
		expectedSize := int(math.Ceil(float64(privKey.Params().BitSize) / 8))

		if len(pubKey) < 1 || pubKey[0] != 0x04 || expectedSize*2 != len(pubKey)-1 {
			return WrongData(), nil
		}

		X.SetBytes(pubKey[1 : expectedSize+1])
		Y.SetBytes(pubKey[1+expectedSize:])

		plaintext = privKey.DecryptShared(X, Y)
		// pad to match expected size
		plaintext = append(make([]byte, expectedSize-len(plaintext)), plaintext...)
	default:
		log.Printf("invalid private key for PSO:DEC")
		return CardKeyNotSupported(), nil
	}

	if err != nil {
		log.Printf("PSO:DEC error, %v", err)
		return UnrecoverableError(), nil
	}

	log.Printf("PSO:DEC successful")

	return CommandCompleted(plaintext), nil
}

// Encipher implements
// p68, 7.2.12 PSO: ENCIPHER, OpenPGP application Version 3.4.
func (card *Interface) Encipher(data []byte) (rapdu *apdu.RAPDU, err error) {
	// PSO:ENC does not use any OpenPGP key but we still use the decryption
	// subkey for cardholder authentication.
	subkey := card.Dec

	if subkey.PrivateKey.Encrypted {
		return SecurityConditionNotSatisfied(), nil
	}

	rapdu, err = encipher(data)

	if err == nil {
		log.Printf("PSO:ENC successful")
	}

	return
}

// GetChallenge implements
// p74, 7.2.15 GET CHALLENGE, OpenPGP application Version 3.4.
func (card *Interface) GetChallenge(n int) (rapdu *apdu.RAPDU, err error) {
	buf := make([]byte, n)
	_, err = rand.Read(buf)

	if err != nil {
		return UnrecoverableError(), err
	}

	return CommandCompleted(buf), nil
}

// EncryptOFB performs symmetric AES encryption using AES-256-OFB. The
// initialization vector is prepended to the encrypted file, the HMAC for
// authentication is appended: `iv (16 bytes) || ciphertext || hmac (32
// bytes)`.
func EncryptOFB(key []byte, iv []byte, input []byte) (output []byte, err error) {
	block, err := aes.NewCipher(key)

	if err != nil {
		return
	}

	output = iv

	mac := hmac.New(sha256.New, key)
	mac.Write(iv)

	stream := cipher.NewOFB(block, iv)
	output = append(output, make([]byte, len(input))...)

	stream.XORKeyStream(output[len(iv):], input)
	mac.Write(output[len(iv):])

	output = append(output, mac.Sum(nil)...)

	return
}

// DecryptOFB performs symmetric AES decryption using AES-256-OFB. The
// ciphertext format is expected to have the initialization vector prepended
// and an HMAC for authentication appended: `iv (16 bytes) || ciphertext ||
// hmac (32 bytes)`.
func DecryptOFB(key []byte, iv []byte, input []byte) (output []byte, err error) {
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

// Pad implements PKCS7 compliant padding for symmetric AES operation.
func Pad(buf []byte, extraBlock bool) []byte {
	padLen := 0
	r := len(buf) % aes.BlockSize

	if r != 0 {
		padLen = aes.BlockSize - r
	} else if extraBlock {
		padLen = aes.BlockSize
	}

	padding := []byte{(byte)(padLen)}
	padding = bytes.Repeat(padding, padLen)
	buf = append(buf, padding...)

	return buf
}
