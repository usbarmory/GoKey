// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package icc

import (
	"log"

	"github.com/hsanjuan/go-nfctype4/apdu"
	"github.com/keybase/go-crypto/openpgp"
)

const (
	PW_VERIFY = 0x00
	PW_LOCK   = 0xff

	// PW1 only valid for one PSO:CDS command
	PW1_CDS_MULTI = 0x00

	PW1_CDS = 0x81
	PW1     = 0x82
	PW3     = 0x83
)

// Verify implements
// p51, 7.2.2 VERIFY, OpenPGP application Version 3.4.
//
// Unlike most smartcards, in this implementation PW1 represents the actual
// private key passphrase and it is used to decrypt the selected OpenPGP
// private subkey.
//
// Therefore the passphrase/PIN verification status matches the presence of a
// decrypted subkey in memory.
//
// Verification of the admin password (PW3) is not supported as, in this
// implementation, card personalization is managed outside OpenPGP
// specifications.
func (card *Interface) Verify(P1 byte, P2 byte, passphrase []byte) (rapdu *apdu.RAPDU, err error) {
	var subkey *openpgp.Subkey

	defer card.signalVerificationStatus()

	switch P2 {
	case PW1_CDS:
		subkey = card.Sig
	case PW1:
		// Used for PSO:DEC and PSO:ENC, the latter operation does not
		// use any OpenPGP key but we still use the same subkey for
		// cardholder authentication.
		subkey = card.Dec
	case PW3:
		// PW3 is not implemented as card personalization is managed
		// outside OpenPGP specifications.
		return CommandNotAllowed(), nil
	}

	if subkey == nil || subkey.PrivateKey == nil {
		log.Printf("missing key for VERIFY")
		return CommandNotAllowed(), nil
	}

	switch P1 {
	case PW_VERIFY:
		if len(passphrase) == 0 {
			// return access status when PW empty
			if !subkey.PrivateKey.Encrypted {
				return CommandCompleted(nil), nil
			} else {
				return VerifyFail(card.errorCounterPW1), nil
			}
		}

		if !subkey.PrivateKey.Encrypted {
			// To support the out-of-band `unlock` management
			// command over SSH we deviate from specifications.
			//
			// If the key is already decrypted then we return
			// success rather than re-verifying the passphrase.
			//
			// This prevents plaintext transmission of the passphrase
			// (which can be a dummy if already unlocked).
			log.Printf("VERIFY: % X already unlocked", subkey.PrivateKey.Fingerprint)
			return CommandCompleted(nil), nil
		}

		if card.errorCounterPW1 == 0 {
			return VerifyFail(card.errorCounterPW1), nil
		}

		if err = subkey.PrivateKey.Decrypt(passphrase); err == nil {
			log.Printf("VERIFY: % X unlocked", subkey.PrivateKey.Fingerprint)
		}
	case PW_LOCK:
		if subkey.PrivateKey.Encrypted {
			log.Printf("VERIFY: % X already locked", subkey.PrivateKey.Fingerprint)
		} else {
			subkey.PrivateKey = card.Restore(subkey)

			if subkey.PrivateKey.Encrypted {
				log.Printf("VERIFY: % X locked", subkey.PrivateKey.Fingerprint)
			} else {
				log.Printf("VERIFY: % X remains unlocked (no passphrase)", subkey.PrivateKey.Fingerprint)
			}
		}
	default:
		return CommandNotAllowed(), nil
	}

	if err != nil {
		log.Printf("VERIFY: % X unlock error", subkey.PrivateKey.Fingerprint)
		card.errorCounterPW1 -= 1
		return VerifyFail(card.errorCounterPW1), nil
	}

	return CommandCompleted(nil), nil
}

func (card *Interface) signalVerificationStatus() {
	for _, subkey := range []*openpgp.Subkey{card.Sig, card.Dec} {
		if subkey != nil && subkey.PrivateKey != nil && subkey.PrivateKey.PrivateKey != nil && !subkey.PrivateKey.Encrypted {
			// at least one key is unlocked
			LED("blue", true)
			return
		}
	}

	// all keys are locked
	LED("blue", false)
}
