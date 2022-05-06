// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package icc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"regexp"

	"github.com/usbarmory/GoKey/internal/snvs"

	"github.com/hsanjuan/go-nfctype4/apdu"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

// Diversifier for hardware key derivation (OpenPGP key wrapping).
const DiversifierPGP = "GoKeySNVSOpenPGP"

const (
	// p48, 7.1 Usage of ISO Standard Commands, OpenPGP application Version 3.4.
	SELECT                       = 0xa4
	GET_DATA                     = 0xca
	VERIFY                       = 0x20
	PUT_DATA_1                   = 0xda
	PUT_DATA_2                   = 0xdb
	GENERATE_ASYMMETRIC_KEY_PAIR = 0x47
	GET_CHALLENGE                = 0x84
	PERFORM_SECURITY_OPERATION   = 0x2a

	// Not implemented:
	//   SELECT DATA
	//   GET NEXT DATA
	//   CHANGE REFERENCE DATA
	//   RESET RETRY COUNTER
	//   INTERNAL AUTHENTICATE
	//   GET RESPONSE
	//   TERMINATE DF
	//   ACTIVATE FILE
	//   MANAGE SECURITY ENVIRONMENT

	// Security Operations
	COMPUTE_DIGITAL_SIGNATURE = 0x9e9a
	DECIPHER                  = 0x8086
	ENCIPHER                  = 0x8680

	DEFAULT_PW1_ERROR_COUNTER = 3
)

// Interface implements an OpenPGP card instance.
type Interface struct {
	// Unique serial number
	Serial [4]byte
	// p30, 4.4.3.3 Name, OpenPGP application Version 3.4
	Name string
	// p30, 4.4.3.4 Name, OpenPGP application Version 3.4
	Language string
	// p31, 4.4.3.5 Name, OpenPGP application Version 3.4
	Sex string

	URL       string
	LoginData []byte

	// enable APDU debugging
	Debug bool
	// enable device unique hardware encryption for bundled private keys
	SNVS bool

	// Armored secret key
	ArmoredKey []byte
	// Secret key
	Key *openpgp.Entity
	// Signature subkey
	Sig *openpgp.Subkey
	// Decryption subkey
	Dec *openpgp.Subkey
	// Authentication subkey
	Aut *openpgp.Subkey

	// encrypted private key caches for PW_LOCK
	sig packet.PrivateKey
	dec packet.PrivateKey
	aut packet.PrivateKey

	// currently unused
	CA []*openpgp.Entity

	// volatile (TODO: make it permanent)
	errorCounterPW1 uint8
	// no reset functionality, unused and fixed to 0x00
	errorCounterRC uint8
	// PW3 PIN is not supported, unused and fixed to 0x00
	errorCounterPW3 uint8
	// volatile (TODO: make it permanent)
	digitalSignatureCounter uint32

	// internal state flags
	selected    bool
	initialized bool
}

// Init initializes the OpenPGP card instance, using passed amored secret key
// material.
//
// The SNVS argument indicates whether private keys (which are already
// encrypted with the passphrase unless the user created them without one) are
// to be stored encrypted at rest with a device specific hardware derived key.
func (card *Interface) Init() (err error) {
	if card.initialized {
		return errors.New("card already initialized")
	}

	if card.SNVS {
		card.ArmoredKey, err = snvs.Decrypt(card.ArmoredKey, []byte(DiversifierPGP))
	}

	if err != nil {
		return fmt.Errorf("OpenPGP key decryption failed, %v", err)
	}

	card.Key, err = decodeArmoredKey([]byte(card.ArmoredKey))

	if err != nil {
		return fmt.Errorf("OpenPGP key decoding failed, %v", err)
	}

	card.Sig, card.Dec, card.Aut = decodeSubkeys(card.Key)

	// cache encrypted private keys for PW_LOCK
	if card.Sig != nil && card.Sig.PrivateKey != nil {
		card.sig = *card.Sig.PrivateKey
	}
	if card.Dec != nil && card.Dec.PrivateKey != nil {
		card.dec = *card.Dec.PrivateKey
	}
	if card.Aut != nil && card.Aut.PrivateKey != nil {
		card.aut = *card.Aut.PrivateKey
	}

	card.errorCounterPW1 = DEFAULT_PW1_ERROR_COUNTER
	card.initialized = true

	log.Printf("OpenPGP card initialized")
	log.Print(card.Status())

	LED("white", false)
	LED("blue", false)

	card.signalVerificationStatus()

	return
}

// Initialized returns the OpenPGP card initialization state.
func (card *Interface) Initialized() bool {
	return card.initialized
}

// Restore overwrites decrypted subkeys with their encrypted version, imported
// at card initialization.
func (card *Interface) Restore(subkey *openpgp.Subkey) *packet.PrivateKey {
	if subkey == nil || subkey.PrivateKey == nil {
		return nil
	}

	for _, privateKey := range []packet.PrivateKey{card.sig, card.dec, card.aut} {
		if bytes.Equal(privateKey.Fingerprint, subkey.PrivateKey.Fingerprint) {
			return &privateKey
		}
	}

	return nil
}

// Select implements
// p50, 7.2.1 SELECT, OpenPGP application Version 3.4.
func (card *Interface) Select(file []byte) (rapdu *apdu.RAPDU, _ error) {
	rapdu = FileNotFound()

	if bytes.Equal(file, RID) || bytes.Equal(file, card.AID()) {
		rapdu = CommandCompleted(nil)
		card.selected = true
	} else if card.selected {
		// A SELECT for a different application sets the status to 'not
		// verified' for all PWs.
		_, _ = card.Verify(PW_LOCK, PW1_CDS, nil)
		_, _ = card.Verify(PW_LOCK, PW1, nil)
		_, _ = card.Verify(PW_LOCK, PW3, nil)
		card.selected = false
	}

	return
}

// RawCommand parses a buffer representing an APDU command and redirects it to
// the relevant handler. A buffer representing the APDU response is returned.
func (card *Interface) RawCommand(buf []byte) ([]byte, error) {
	capdu := &apdu.CAPDU{}
	_, err := capdu.Unmarshal(buf)

	if err != nil {
		return nil, err
	}

	rapdu, err := card.Command(capdu)

	if err != nil {
		return nil, err
	}

	return rapdu.Marshal()
}

// Command parses an APDU command and redirects it to the relevant handler. An
// APDU response is returned.
func (card *Interface) Command(capdu *apdu.CAPDU) (rapdu *apdu.RAPDU, err error) {
	rapdu = CommandNotAllowed()

	if card.Debug {
		log.Printf("<< %+v", capdu)
	}

	if capdu.CLA != 0x00 {
		return
	}

	params := binary.BigEndian.Uint16([]byte{capdu.P1, capdu.P2})

	// p48, 7.1 Usage of ISO Standard Commands, OpenPGP application Version 3.4
	switch capdu.INS {
	case SELECT:
		rapdu, err = card.Select(capdu.Data)
	case GET_DATA:
		rapdu, err = card.GetData(params)
	case VERIFY:
		rapdu, err = card.Verify(capdu.P1, capdu.P2, capdu.Data)
	case PUT_DATA_1, PUT_DATA_2:
		rapdu, err = card.PutData(params)
	case GENERATE_ASYMMETRIC_KEY_PAIR:
		rapdu, err = card.GenerateAsymmetricKeyPair(params, capdu.Data)
	case GET_CHALLENGE:
		rapdu, err = card.GetChallenge(int(capdu.GetLe()))
	case PERFORM_SECURITY_OPERATION:
		LED("white", true)
		defer LED("white", false)

		switch params {
		case COMPUTE_DIGITAL_SIGNATURE:
			rapdu, err = card.ComputeDigitalSignature(capdu.Data)
		case DECIPHER:
			rapdu, err = card.Decipher(capdu.Data)
		case ENCIPHER:
			rapdu, err = card.Encipher(capdu.Data)
		default:
			log.Printf("unsupported PSO %x", params)
		}
	default:
		log.Printf("unsupported INS %x", capdu.INS)
	}

	if rapdu == nil {
		rapdu = CommandNotAllowed()
	}

	if card.Debug {
		log.Printf(">> %+v", rapdu)
	}

	return
}

// Status returns card key fingerprints and encryption status in textual
// format.
func (card *Interface) Status() string {
	var status bytes.Buffer

	status.WriteString("---------------------------------------------------- OpenPGP smartcard ----\n")
	status.WriteString(fmt.Sprintf("Initialized ............: %v\n", card.initialized))
	status.WriteString(fmt.Sprintf("Secure storage .........: %v\n", card.SNVS))
	status.WriteString(fmt.Sprintf("Serial number ..........: %X\n", card.Serial))
	status.WriteString(fmt.Sprintf("Digital signature count.: %v\n", card.digitalSignatureCounter))
	status.WriteString("Secret key .............: ")

	r := regexp.MustCompile(`([[:xdigit:]]{4})`)

	if k := card.Key; k != nil {
		fp := fmt.Sprintf("%X\n", k.PrimaryKey.Fingerprint)
		status.WriteString(r.ReplaceAllString(fp, "$1 "))
	} else {
		status.WriteString("missing\n")
	}

	desc := []string{"Signature subkey .......: ", "Decryption subkey ......: ", "Authentication subkey ..: "}

	for i, sk := range []*openpgp.Subkey{card.Sig, card.Dec, card.Aut} {
		status.WriteString(desc[i])

		if sk != nil {
			fp := fmt.Sprintf("%X\n", sk.PublicKey.Fingerprint)
			status.WriteString(r.ReplaceAllString(fp, "$1 "))

			if pk := sk.PrivateKey; pk != nil {
				status.WriteString(fmt.Sprintf("               encrypted: %v\n", pk.Encrypted))
			}
		} else {
			status.WriteString("missing\n")
		}
	}

	return status.String()
}
