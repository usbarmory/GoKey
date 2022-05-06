// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package icc

import (
	"log"

	"github.com/hsanjuan/go-nfctype4/apdu"
	"github.com/ProtonMail/go-crypto/openpgp"
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

	var msg string

	switch P1 {
	case PW_VERIFY:
		if len(passphrase) == 0 {
			// return access status when PW empty
			if !subkey.PrivateKey.Encrypted {
				rapdu = CommandCompleted(nil)
			} else {
				rapdu = VerifyFail(card.errorCounterPW1)
			}
		} else if !subkey.PrivateKey.Encrypted {
			// To support the out-of-band `unlock` management
			// command over SSH we deviate from specifications.
			//
			// If the key is already decrypted then we return
			// success rather than re-verifying the passphrase.
			//
			// This prevents plaintext transmission of the passphrase
			// (which can be a dummy if already unlocked).
			msg = "already unlocked"
		} else if card.errorCounterPW1 == 0 {
			// for now this counter is volatile across reboots
			msg = "error counter blocked, cannot unlock"
			rapdu = VerifyFail(card.errorCounterPW1)
		} else if subkey.PrivateKey.Decrypt(passphrase) == nil {
			// correct verification sets resets counter to default value
			card.errorCounterPW1 = DEFAULT_PW1_ERROR_COUNTER
			msg = "unlocked"
		} else {
			// The standard is not clear on the specific conditions
			// that decrese the counter as "incorrect usage" is
			// mentioned. This implementation only cares to prevent
			// passphrase brute forcing.
			card.errorCounterPW1 -= 1
			msg = "unlock error"
			rapdu = VerifyFail(card.errorCounterPW1)
		}
	case PW_LOCK:
		if subkey.PrivateKey.Encrypted {
			msg = "already locked"
		} else {
			subkey.PrivateKey = card.Restore(subkey)

			if subkey.PrivateKey.Encrypted {
				msg = "locked"
			} else {
				msg = "remains unlocked (no passphrase)"
			}
		}
	default:
		return CommandNotAllowed(), nil
	}

	if msg != "" {
		log.Printf("VERIFY: % X %s", subkey.PrivateKey.Fingerprint, msg)
	}

	if rapdu == nil {
		rapdu = CommandCompleted(nil)
	}

	return
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
