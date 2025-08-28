// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package icc

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"log"

	"github.com/ProtonMail/go-crypto/openpgp/ecdh"
	"github.com/ProtonMail/go-crypto/openpgp/ecdsa"
	"github.com/hsanjuan/go-nfctype4/apdu"
)

const (
	// p65, 7.2.11 PSO: DECIPHER, OpenPGP application Version 3.4
	RSA_PADDING = 0x00
	AES_PADDING = 0x02
)

func padToKeySize(pub ecdsa.PublicKey, b []byte) []byte {
	// RFC 4880 - OpenPGP Message Format:
	// The size of an MPI is ((MPI.length + 7) / 8) + 2 octets.
	k := ((pub.X.BitLen() + 7) / 8) + 2
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

		plaintext, err = privKey.Decrypt(rand.Reader, data[1:], nil)
	case *ecdh.PrivateKey:
		if data[0] != DO_CIPHER {
			log.Printf("invalid private key for PSO:DEC")
			return CardKeyNotSupported(), nil
		}

		// p66, 7.2.11 PSO: DECIPHER, OpenPGP application Version 3.4
		pubKey := v(v(v(data, DO_CIPHER), DO_PUB_KEY), DO_EXT_PUB_KEY)
		expectedSize := (len(pubKey) - 1) / 2

		if len(pubKey) < 1 || pubKey[0] != 0x04 || expectedSize*2 != len(pubKey)-1 {
			return WrongData(), nil
		}

		plaintext, err = privKey.GetCurve().Decaps(pubKey, privKey.D)
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

	if rapdu, err = encipher(data); err == nil {
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
